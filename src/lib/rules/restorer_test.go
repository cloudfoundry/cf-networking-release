package rules_test

import (
	"fmt"
	"lib/filelock"
	"lib/rules"
	"lib/testsupport"
	"os/exec"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Restorer", func() {
	var (
		restorer *rules.Restorer
		locker   *filelock.Locker
	)
	BeforeEach(func() {
		locker = &filelock.Locker{
			Path: "/tmp/restorer.lock",
		}
		restorer = &rules.Restorer{
			Locker: locker,
		}
	})
	It("Writes IP tables rules", func() {
		onlyRunOnLinux()
		err := restorer.Restore(restoreInput(10000))
		Expect(err).NotTo(HaveOccurred())
		Expect(AllIPTablesRules("filter")).To(ContainElement(`-N chain-10000`))
	})
	It("supports concurrent writes", func() {
		onlyRunOnLinux()
		rules := []string{}
		numRules := 100
		numWorkers := 10
		for i := 0; i < numRules; i++ {
			rules = append(rules, restoreInput(i))
		}
		runner := testsupport.ParallelRunner{
			NumWorkers: numWorkers,
		}
		restoreWorker := func(item string) {
			err := restorer.Restore(item)
			Expect(err).NotTo(HaveOccurred())
		}
		runner.RunOnSliceStrings(rules, restoreWorker)

		for i := 0; i < numRules; i++ {
			Expect(AllIPTablesRules("filter")).To(ContainElement(fmt.Sprintf("-N chain-%d", i)))
		}
	})
	It("supports concurrent writes with iptables if that uses the same lock", func() {
		onlyRunOnLinux()
		rules := []string{}
		numRules := 100
		numWorkers := 10
		for i := 0; i < numRules; i++ {
			rules = append(rules, fmt.Sprintf("%d", i))
		}
		runner := testsupport.ParallelRunner{
			NumWorkers: numWorkers,
		}
		restoreWorker := func(item string) {
			input := fmt.Sprintf("*filter\n-N rst2-%s\nCOMMIT\n", item)
			err := restorer.Restore(input)
			Expect(err).NotTo(HaveOccurred())
			f, err := locker.Open()
			if err != nil {
				panic(err)
			}
			defer f.Close()
			err = exec.Command("iptables", "-N", fmt.Sprintf("ipt-%s", item)).Run()
			Expect(err).NotTo(HaveOccurred())
		}
		runner.RunOnSliceStrings(rules, restoreWorker)

		for i := 0; i < numRules; i++ {
			Expect(AllIPTablesRules("filter")).To(ContainElement(fmt.Sprintf("-N rst2-%d", i)))
			Expect(AllIPTablesRules("filter")).To(ContainElement(fmt.Sprintf("-N ipt-%d", i)))
		}
	})

	It("supports concurrent writes while iptables reads", func() {
		onlyRunOnLinux()
		rules := []string{}
		numRules := 100
		numWorkers := 10
		for i := 0; i < numRules; i++ {
			rules = append(rules, fmt.Sprintf("%d", i))
		}
		runner := testsupport.ParallelRunner{
			NumWorkers: numWorkers,
		}
		restoreWorker := func(item string) {
			input := fmt.Sprintf("*filter\n-N rst3-%s\nCOMMIT\n", item)
			err := restorer.Restore(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(exec.Command("iptables", "-w", "-S").Run()).To(Succeed())
		}
		runner.RunOnSliceStrings(rules, restoreWorker)

		for i := 0; i < numRules; i++ {
			Expect(AllIPTablesRules("filter")).To(ContainElement(fmt.Sprintf("-N rst3-%d", i)))
		}
	})

})

func onlyRunOnLinux() {
	if runtime.GOOS != "linux" {
		Skip("OS is not linux. Skipping...")
	}
}

func AllIPTablesRules(tableName string) []string {
	iptablesSession, err := gexec.Start(exec.Command("iptables", "-w", "-S", "-t", tableName), nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Eventually(iptablesSession).Should(gexec.Exit(0))
	return strings.Split(strings.TrimSpace(string(iptablesSession.Out.Contents())), "\n")
}

func restoreInput(index int) string {
	return fmt.Sprintf("*filter\n-N chain-%d\nCOMMIT\n", index)
}
