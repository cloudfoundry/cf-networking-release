package rules

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pivotal-golang/lager"
)

type Timestamper struct{}

func (_ Timestamper) CurrentTime() int {
	return int(time.Now().Unix())
}

//go:generate counterfeiter -o ../fakes/iptables.go --fake-name IPTables . IPTables
type IPTables interface {
	Exists(table, chain string, rulespec ...string) (bool, error)
	Insert(table, chain string, pos int, rulespec ...string) error
	AppendUnique(table, chain string, rulespec ...string) error
	Delete(table, chain string, rulespec ...string) error
	List(table, chain string) ([]string, error)
	NewChain(table, chain string) error
	ClearChain(table, chain string) error
	DeleteChain(table, chain string) error
}

//go:generate counterfeiter -o ../fakes/timestamper.go --fake-name TimeStamper . TimeStamper
type TimeStamper interface {
	CurrentTime() int
}

type Enforcer struct {
	Logger      lager.Logger
	timestamper TimeStamper
	iptables    IPTables
}

func NewEnforcer(logger lager.Logger, timestamper TimeStamper, ipt IPTables) *Enforcer {
	return &Enforcer{
		Logger:      logger,
		timestamper: timestamper,
		iptables:    ipt,
	}
}

func (e *Enforcer) Enforce(chainPrefix string, rules []Rule) error {
	newTime := e.timestamper.CurrentTime()
	chain := fmt.Sprintf("%s%d", chainPrefix, newTime)

	err := e.iptables.NewChain("filter", chain)
	if err != nil {
		e.Logger.Error("create-chain", err)
		return fmt.Errorf("creating chain: %s", err)
	}

	for _, rule := range rules {
		err = rule.Enforce(chain, e.iptables, e.Logger)
		if err != nil {
			return err
		}
	}

	err = e.iptables.Insert("filter", "FORWARD", 1, []string{"-j", chain}...)
	if err != nil {
		e.Logger.Error("insert-chain", err)
		return fmt.Errorf("inserting chain: %s", err)
	}

	err = e.cleanupOldRules(chainPrefix, int(newTime))
	if err != nil {
		e.Logger.Error("cleanup-rules", err)
		return err
	}

	return nil
}

func (e *Enforcer) cleanupOldRules(chainPrefix string, newTime int) error {
	chainList, err := e.iptables.List("filter", "FORWARD")
	if err != nil {
		return fmt.Errorf("listing forward rules: %s", err)
	}

	re := regexp.MustCompile(chainPrefix + "[0-9]{10}")
	for _, c := range chainList {
		timeStampedChain := string(re.Find([]byte(c)))

		if timeStampedChain != "" {
			oldTime, err := strconv.Atoi(strings.TrimPrefix(timeStampedChain, chainPrefix))
			if err != nil {
				return err // not tested
			}

			if oldTime < newTime {
				err = e.cleanupOldChain(timeStampedChain)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (e *Enforcer) cleanupOldChain(timeStampedChain string) error {
	err := e.iptables.Delete("filter", "FORWARD", []string{"-j", timeStampedChain}...)
	if err != nil {
		return fmt.Errorf("cleanup old chain: %s", err)
	}

	err = e.iptables.ClearChain("filter", timeStampedChain)
	if err != nil {
		return fmt.Errorf("cleanup old chain: %s", err)
	}

	err = e.iptables.DeleteChain("filter", timeStampedChain)
	if err != nil {
		return fmt.Errorf("cleanup old chain: %s", err)
	}

	return nil
}
