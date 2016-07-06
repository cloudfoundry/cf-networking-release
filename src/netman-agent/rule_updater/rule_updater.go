package rule_updater

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

func setupDefaultIptablesChain(ipt iptables) error {
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
		"-i", "cni-flannel0", "-j", "netman--forward-default",
	}...)
	if err != nil {
		return err
	}

	err = ipt.AppendUnique("filter", "netman--forward-default", []string{
		"-m", "state", "--state", "ESTABLISHED,RELATED",
		"-j", "ACCEPT",
	}...)
	if err != nil {
		return err
	}

	err = ipt.AppendUnique("filter", "netman--forward-default", []string{
		"-j", "DROP",
	}...)
	if err != nil {
		return err
	}

	return nil
}

func New(logger lager.Logger, storeReader storeReader, policyClient policyClient, iptables iptables, vni int) (*Updater, error) {
	err := setupDefaultIptablesChain(iptables)
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
	containers := u.storeReader.GetContainers()
	policies, err := u.policyClient.GetPolicies()
	if err != nil {
		u.Logger.Error("get-policies", err)
		return fmt.Errorf("get policies failed: %s", err)
	}

	newTime := time.Now().Unix()
	newChain := fmt.Sprintf("netman--forward-%d", newTime)
	err = u.iptables.NewChain("filter", newChain)
	if err != nil {
		u.Logger.Error("create-chain", err)
		return fmt.Errorf("creating chain: %s", err)
	}

	for _, policy := range policies {
		srcContainers, srcOk := containers[policy.Source.ID]
		dstContainers, dstOk := containers[policy.Destination.ID]

		// local dest
		if dstOk {
			for _, dstContainer := range dstContainers {
				u.Logger.Info("enforce-remote-rule", lager.Data{
					"srcTag": policy.Source.Tag,
					"dstIP":  dstContainer.IP,
					"port":   policy.Destination.Port,
					"proto":  policy.Destination.Protocol,
					"vni":    u.VNI,
				})
			}
		}

		if srcOk {
			for _, srcContainer := range srcContainers {
				u.Logger.Info("set-local-tag", lager.Data{
					"srcTag": policy.Source.Tag,
					"srcIP":  srcContainer.IP,
				})
			}
		}

		// local
		if srcOk && dstOk {
			for _, srcContainer := range srcContainers {
				for _, dstContainer := range dstContainers {
					err = u.iptables.AppendUnique("filter", newChain, []string{
						"-i", "cni-flannel0",
						"-s", srcContainer.IP,
						"-d", dstContainer.IP,
						"-p", policy.Destination.Protocol,
						"--dport", strconv.Itoa(policy.Destination.Port),
						"-j", "ACCEPT",
					}...)
					if err != nil {
						u.Logger.Error("append-rule", err)
						return fmt.Errorf("appending rule: %s", err)
					}

					u.Logger.Info("enforce-local-rule", lager.Data{
						"srcIP": srcContainer.IP,
						"dstIP": dstContainer.IP,
						"port":  policy.Destination.Port,
						"proto": policy.Destination.Protocol,
					})
				}
			}
		}
	}

	err = u.iptables.Insert("filter", "FORWARD", 1, []string{
		"-i", "cni-flannel0", "-j", newChain,
	}...)
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
	err := u.iptables.Delete("filter", "FORWARD", []string{
		"-i", "cni-flannel0", "-j", timeStampedChain,
	}...)
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
