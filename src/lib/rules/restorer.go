package rules

import (
	"fmt"
	"lib/filelock"
	"log"
	"os/exec"
	"strings"
)

type Restorer struct {
	Locker *filelock.Locker
}

func (r *Restorer) Restore(input string) error {
	cmd := exec.Command("iptables-restore", "--noflush")
	cmd.Stdin = strings.NewReader(input)

	f, err := r.Locker.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	bytes, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("iptables-restore error: %s combined output: %s", err, string(bytes))
	}
	return nil
}
