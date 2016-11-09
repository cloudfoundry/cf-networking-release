package rules

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

type LockedIPTables struct {
	IPTables IPTables
}

func (l *LockedIPTables) Exists(table, chain string, rulespec ...string) (bool, error) {
	return l.IPTables.Exists(table, chain, rulespec...)
}
func (l *LockedIPTables) Insert(table, chain string, pos int, rulespec ...string) error {
	return l.IPTables.Insert(table, chain, pos, rulespec...)
}
func (l *LockedIPTables) AppendUnique(table, chain string, rulespec ...string) error {
	return l.IPTables.AppendUnique(table, chain, rulespec...)
}
func (l *LockedIPTables) Delete(table, chain string, rulespec ...string) error {
	return l.IPTables.Delete(table, chain, rulespec...)
}
func (l *LockedIPTables) List(table, chain string) ([]string, error) {
	return l.IPTables.List(table, chain)
}
func (l *LockedIPTables) NewChain(table, chain string) error {
	return l.IPTables.NewChain(table, chain)
}
func (l *LockedIPTables) ClearChain(table, chain string) error {
	return l.IPTables.ClearChain(table, chain)
}
func (l *LockedIPTables) DeleteChain(table, chain string) error {
	return l.IPTables.DeleteChain(table, chain)
}
