package rules

import (
	"os/exec"
	"strings"
)

type Restorer struct {
}

// untested
func (r *Restorer) Restore(input string) error {
	cmd := exec.Command("iptables-restore", "--noflush")
	cmd.Stdin = strings.NewReader(input)

	return cmd.Run()
}
