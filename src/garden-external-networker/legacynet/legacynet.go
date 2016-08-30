package legacynet

import "fmt"

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
