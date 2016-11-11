package legacynet

import (
	"errors"
	"fmt"
)

//go:generate counterfeiter -o ../fakes/chain_namer.go --fake-name ChainNamer . chainNamer
type chainNamer interface {
	Prefix(prefix, body string) string
	Postfix(body, suffix string) (string, error)
}

type ChainNamer struct {
	MaxLength int
}

func (n *ChainNamer) truncate(name string, length int) string {
	if len(name) > length {
		name = name[:length]
	}
	return name
}

func (n *ChainNamer) Prefix(prefix, body string) string {
	return n.truncate(fmt.Sprintf("%s--%s", prefix, body), n.MaxLength)
}

func (n *ChainNamer) Postfix(body, suffix string) (string, error) {
	newBodyLen := n.MaxLength - len("--"+suffix)
	if newBodyLen < 0 {
		return "", errors.New("suffix too long, string could not be truncated to max length")
	}
	return fmt.Sprintf("%s--%s", n.truncate(body, newBodyLen), suffix), nil
}
