package main

import (
	"fmt"
	"os"
	"strconv"

	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
)

func main() {
	if err := mainWithError(); err != nil {
		os.Stderr.Write([]byte(err.Error() + "\n"))
		os.Exit(1)
	}
}

func mainWithError() error {
	args := os.Args
	if len(args) < 2 {
		return fmt.Errorf("usage: requires number of existing asgs")
	}

	numExistingASGs, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("parsing num existing asgs: %s", err)
	}

	asg := testsupport.BuildASG(numExistingASGs)
	os.Stdout.Write([]byte(asg))

	return nil
}
