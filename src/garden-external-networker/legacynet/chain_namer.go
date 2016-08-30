package legacynet

import "fmt"

//go:generate counterfeiter -o ../fakes/chain_namer.go --fake-name ChainNamer . chainNamer
type chainNamer interface {
	Name(prefix, containerHandle string) string
}

type ChainNamer struct {
	MaxLength int
}

func (n *ChainNamer) truncate(name string) string {
	if len(name) > n.MaxLength {
		name = name[:n.MaxLength]
	}
	return name
}

func (n *ChainNamer) Name(prefix, containerHandle string) string {
	return n.truncate(fmt.Sprintf("%s--%s", prefix, containerHandle))
}
