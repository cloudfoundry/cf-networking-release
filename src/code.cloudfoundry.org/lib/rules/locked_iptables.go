package rules

import (
	"fmt"
	"os/exec"
	"strings"
)

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

//go:generate counterfeiter -o ../fakes/iptables_extended.go --fake-name IPTablesAdapter . IPTablesAdapter
type IPTablesAdapter interface {
	Exists(table, chain string, rulespec IPTablesRule) (bool, error)
	Delete(table, chain string, rulespec IPTablesRule) error
	List(table, chain string) ([]string, error)
	NewChain(table, chain string) error
	ClearChain(table, chain string) error
	DeleteChain(table, chain string) error
	BulkInsert(table, chain string, pos int, rulespec ...IPTablesRule) error
	BulkAppend(table, chain string, rulespec ...IPTablesRule) error
}

//go:generate counterfeiter -o ../fakes/locker.go --fake-name Locker . locker
type locker interface {
	Lock() error
	Unlock() error
}

//go:generate counterfeiter -o ../fakes/restorer.go --fake-name Restorer . restorer
type restorer interface {
	Restore(ruleState string) error
}

type Restorer struct{}

func (r *Restorer) Restore(input string) error {
	cmd := exec.Command("iptables-restore", "--noflush")
	cmd.Stdin = strings.NewReader(input)

	bytes, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("iptables-restore error: %s combined output: %s", err, string(bytes))
	}
	return nil
}

type LockedIPTables struct {
	IPTables iptables
	Locker   locker
	Restorer restorer
}

func handleIPTablesError(err1, err2 error) error {
	return fmt.Errorf("iptables call: %+v and unlock: %+v", err1, err2)
}

func (l *LockedIPTables) Exists(table, chain string, rulespec IPTablesRule) (bool, error) {
	if err := l.Locker.Lock(); err != nil {
		return false, fmt.Errorf("lock: %s", err)
	}

	b, err := l.IPTables.Exists(table, chain, rulespec...)
	if err != nil {
		return false, handleIPTablesError(err, l.Locker.Unlock())
	}

	return b, l.Locker.Unlock()
}

func (l *LockedIPTables) bulkAction(table, prefix string, rulespec ...IPTablesRule) error {
	if err := l.Locker.Lock(); err != nil {
		return fmt.Errorf("lock: %s", err)
	}

	input := []string{fmt.Sprintf("*%s\n", table)}
	for _, r := range rulespec {
		tmp := fmt.Sprintf("%s %s\n", prefix, strings.Join(r, " "))
		input = append(input, tmp)
	}
	input = append(input, "COMMIT\n")

	err := l.Restorer.Restore(strings.Join(input, ""))
	if err != nil {
		return handleIPTablesError(err, l.Locker.Unlock())
	}

	return l.Locker.Unlock()
}

func (l *LockedIPTables) BulkInsert(table, chain string, pos int, rulespec ...IPTablesRule) error {
	return l.bulkAction(table, fmt.Sprintf("-I %s %d", chain, pos), rulespec...)
}

func (l *LockedIPTables) BulkAppend(table, chain string, rulespec ...IPTablesRule) error {
	return l.bulkAction(table, fmt.Sprintf("-A %s", chain), rulespec...)
}

func (l *LockedIPTables) Delete(table, chain string, rulespec IPTablesRule) error {
	if err := l.Locker.Lock(); err != nil {
		return fmt.Errorf("lock: %s", err)
	}

	err := l.IPTables.Delete(table, chain, rulespec...)
	if err != nil {
		return handleIPTablesError(err, l.Locker.Unlock())
	}

	return l.Locker.Unlock()
}

func (l *LockedIPTables) List(table, chain string) ([]string, error) {
	if err := l.Locker.Lock(); err != nil {
		return nil, fmt.Errorf("lock: %s", err)
	}

	ret, err := l.IPTables.List(table, chain)
	if err != nil {
		return nil, handleIPTablesError(err, l.Locker.Unlock())
	}

	return ret, l.Locker.Unlock()
}

func (l *LockedIPTables) NewChain(table, chain string) error {
	return l.chainExec(table, chain, l.IPTables.NewChain)
}
func (l *LockedIPTables) ClearChain(table, chain string) error {
	return l.chainExec(table, chain, l.IPTables.ClearChain)
}
func (l *LockedIPTables) DeleteChain(table, chain string) error {
	return l.chainExec(table, chain, l.IPTables.DeleteChain)
}

func (l *LockedIPTables) chainExec(table, chain string, action func(string, string) error) error {
	if err := l.Locker.Lock(); err != nil {
		return fmt.Errorf("lock: %s", err)
	}
	if err := action(table, chain); err != nil {
		return handleIPTablesError(err, l.Locker.Unlock())
	}

	return l.Locker.Unlock()
}
