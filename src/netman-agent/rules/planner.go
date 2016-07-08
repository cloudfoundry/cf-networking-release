package rules

import (
	"fmt"
	"netman-agent/models"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o ../fakes/store_reader.go --fake-name StoreReader . storeReader
type storeReader interface {
	GetContainers() map[string][]models.Container
}

//go:generate counterfeiter -o ../fakes/policy_client.go --fake-name PolicyClient . policyClient
type policyClient interface {
	GetPolicies() ([]models.Policy, error)
}

type Updater struct {
	Logger       lager.Logger
	storeReader  storeReader
	policyClient policyClient
	iptables     iptables
	VNI          int
}

//go:generate counterfeiter -o ../fakes/iptables.go --fake-name IPTables . iptables
type iptables interface {
	Exists(table, chain string, rulespec ...string) (bool, error)
	Insert(table, chain string, pos int, rulespec ...string) error
	AppendUnique(table, chain string, rulespec ...string) error
	Delete(table, chain string, rulespec ...string) error
	List(table, chain string) ([]string, error)
	NewChain(table, chain string) error
	ClearChain(table, chain string) error
	DeleteChain(table, chain string) error
}

func setupDefaultIptablesChain(ipt iptables, localSubnet string, vni int) error {
	rules, err := ipt.List("filter", "FORWARD")
	if err != nil {
		return err
	}
	for _, r := range rules {
		if strings.Contains(r, "netman--forward-default") {
			return nil
		}
	}

	err = ipt.NewChain("filter", "netman--forward-default")
	if err != nil {
		return err
	}

	err = ipt.AppendUnique("filter", "FORWARD", []string{
		"-j", "netman--forward-default",
	}...)
	if err != nil {
		return err
	}

	// default allow for local containers to respond
	err = ipt.AppendUnique("filter", "netman--forward-default", []string{
		"-i", "cni-flannel0",
		"-m", "state", "--state", "ESTABLISHED,RELATED",
		"-j", "ACCEPT",
	}...)
	if err != nil {
		return err
	}

	// default deny for local containers
	err = ipt.AppendUnique("filter", "netman--forward-default", []string{
		"-i", "cni-flannel0",
		"-s", localSubnet,
		"-d", localSubnet,
		"-j", "DROP",
	}...)
	if err != nil {
		return err
	}

	// default allow for remote containers to respond
	err = ipt.AppendUnique("filter", "netman--forward-default", []string{
		"-i", fmt.Sprintf("flannel.%d", vni),
		"-m", "state", "--state", "ESTABLISHED,RELATED",
		"-j", "ACCEPT",
	}...)
	if err != nil {
		return err
	}

	// default deny for remote containers
	err = ipt.AppendUnique("filter", "netman--forward-default", []string{
		"-i", fmt.Sprintf("flannel.%d", vni),
		"-j", "DROP",
	}...)
	if err != nil {
		return err
	}
	return nil
}

func New(logger lager.Logger, storeReader storeReader, policyClient policyClient, iptables iptables, vni int, localSubnet string) (*Updater, error) {
	err := setupDefaultIptablesChain(iptables, localSubnet, vni)
	if err != nil {
		return nil, fmt.Errorf("setting up default chain: %s", err)
	}

	return &Updater{
		Logger:       logger,
		storeReader:  storeReader,
		policyClient: policyClient,
		iptables:     iptables,
		VNI:          vni,
	}, nil
}

func (u *Updater) Update() error {
	rules, err := u.Rules()
	if err != nil {
		return err
	}
	err = u.Enforce(rules)
	if err != nil {
		return err
	}
	return nil
}

func (u *Updater) Rules() ([]Rule, error) {
	containers := u.storeReader.GetContainers()
	policies, err := u.policyClient.GetPolicies()

	rules := []Rule{}

	if err != nil {
		u.Logger.Error("get-policies", err)
		return rules, fmt.Errorf("get policies failed: %s", err)
	}

	for _, policy := range policies {
		srcContainers, srcOk := containers[policy.Source.ID]
		dstContainers, dstOk := containers[policy.Destination.ID]

		// local dest
		if dstOk {
			for _, dstContainer := range dstContainers {
				rules = append(rules, RemoteAllowRule{
					SrcTag:   policy.Source.Tag,
					DstIP:    dstContainer.IP,
					Port:     policy.Destination.Port,
					Proto:    policy.Destination.Protocol,
					VNI:      u.VNI,
					IPTables: u.iptables,
					Logger:   u.Logger,
				})
			}
		}

		if srcOk {
			for _, srcContainer := range srcContainers {
				rules = append(rules, LocalTagRule{
					SourceTag:         policy.Source.Tag,
					SourceContainerIP: srcContainer.IP,
					IPTables:          u.iptables,
					Logger:            u.Logger,
				})
			}
		}

		// local
		if srcOk && dstOk {
			for _, srcContainer := range srcContainers {
				for _, dstContainer := range dstContainers {
					rules = append(rules, LocalAllowRule{
						SrcIP:    srcContainer.IP,
						DstIP:    dstContainer.IP,
						Port:     policy.Destination.Port,
						Proto:    policy.Destination.Protocol,
						IPTables: u.iptables,
						Logger:   u.Logger,
					})
				}
			}
		}
	}

	return rules, nil
}

func (u *Updater) Enforce(rules []Rule) error {
	newTime := time.Now().Unix()
	newChain := fmt.Sprintf("netman--forward-%d", newTime)
	err := u.iptables.NewChain("filter", newChain)
	if err != nil {
		u.Logger.Error("create-chain", err)
		return fmt.Errorf("creating chain: %s", err)
	}

	for _, rule := range rules {
		err = rule.Enforce(rule.Chain(newTime))
		if err != nil {
			return err
		}
	}

	err = u.iptables.Insert("filter", "FORWARD", 1, []string{"-j", newChain}...)
	if err != nil {
		u.Logger.Error("insert-chain", err)
		return fmt.Errorf("inserting chain: %s", err)
	}

	err = u.cleanupOldRules(int(newTime))
	if err != nil {
		u.Logger.Error("cleanup-rules", err)
		return err
	}

	return nil
}

func (u *Updater) cleanupOldRules(newTime int) error {
	chainList, err := u.iptables.List("filter", "FORWARD")
	if err != nil {
		return fmt.Errorf("listing forward rules: %s", err)
	}

	re := regexp.MustCompile("netman--forward-[0-9]{10}")
	for _, c := range chainList {
		timeStampedChain := string(re.Find([]byte(c)))

		if timeStampedChain != "" {
			oldTime, err := strconv.Atoi(strings.TrimPrefix(timeStampedChain, "netman--forward-"))
			if err != nil {
				return err
			}

			if oldTime < newTime {
				err = u.cleanupOldChain(timeStampedChain)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (u *Updater) cleanupOldChain(timeStampedChain string) error {
	err := u.iptables.Delete("filter", "FORWARD", []string{"-j", timeStampedChain}...)
	if err != nil {
		return fmt.Errorf("cleanup old chain: %s", err)
	}

	err = u.iptables.ClearChain("filter", timeStampedChain)
	if err != nil {
		return fmt.Errorf("cleanup old chain: %s", err)
	}

	err = u.iptables.DeleteChain("filter", timeStampedChain)
	if err != nil {
		return fmt.Errorf("cleanup old chain: %s", err)
	}

	return nil
}
