package rules_test

import (
	"fmt"
	"lib/filelock"
	"lib/rules"
	"lib/testsupport"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	goiptables "github.com/coreos/go-iptables/iptables"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Locked IPTables Integration Test", func() {
	var (
		restorer  *rules.Restorer
		locker    *rules.IPTablesLocker
		lockedIPT *rules.LockedIPTables
		ipt       *goiptables.IPTables
	)

	BeforeEach(func() {
		flock := &filelock.Locker{
			Path: "/tmp/restorer.lock",
		}
		locker = &rules.IPTablesLocker{
			FileLocker: flock,
			Mutex:      &sync.Mutex{},
		}
		restorer = &rules.Restorer{}
		var err error
		ipt, err = goiptables.New()
		Expect(err).NotTo(HaveOccurred())
		lockedIPT = &rules.LockedIPTables{
			Locker:   locker,
			Restorer: restorer,
			IPTables: ipt,
		}
	})

	It("bulk inserts iptables rules", func() {
		onlyRunOnLinux()
		err := lockedIPT.BulkInsert("filter", "FORWARD", 1, []rules.GenericRule{
			rules.NewMarkSetRule("1.2.3.4", "A", fmt.Sprintf("guid-%d", 1)),
		}...)
		Expect(err).NotTo(HaveOccurred())
		Expect(AllIPTablesRules("filter")).To(ContainElement("-A FORWARD -s 1.2.3.4/32 -m comment --comment \"src:guid-1\" -j MARK --set-xmark 0xa/0xffffffff"))
	})

	It("bulk appends iptables rules", func() {
		onlyRunOnLinux()
		err := lockedIPT.BulkAppend("filter", "FORWARD", []rules.GenericRule{
			rules.NewMarkAllowRule("1.2.3.4", "tcp", 1234, "A", "some-src-app-guid", "some-dst-app-guid"),
		}...)
		Expect(err).NotTo(HaveOccurred())
		Expect(AllIPTablesRules("filter")).To(ContainElement(
			`-A FORWARD -d 1.2.3.4/32 -p tcp -m tcp --dport 1234 -m mark --mark 0xa -m comment --comment "src:some-src-app-guid_dst:some-dst-app-guid" -j ACCEPT`,
		))
	})

	It("supports concurrent bulk inserts", func() {
		onlyRunOnLinux()
		genericRules := []interface{}{}
		numRules := 100
		numWorkers := 10
		for i := 0; i < numRules; i++ {
			genericRules = append(genericRules, []rules.GenericRule{rules.NewMarkSetRule("1.2.3.4", "A", fmt.Sprintf("guid-%d", i))})
		}
		runner := testsupport.ParallelRunner{
			NumWorkers: numWorkers,
		}
		restoreWorker := func(item interface{}) {
			ruleItems := item.([]rules.GenericRule)
			err := lockedIPT.BulkInsert("filter", "FORWARD", 1, ruleItems...)
			Expect(err).NotTo(HaveOccurred())
		}
		runner.RunOnSlice(genericRules, restoreWorker)
		allRules := AllIPTablesRules("filter")
		for i := 0; i < numRules; i++ {
			Expect(allRules).To(ContainElement(
				fmt.Sprintf("-A FORWARD -s 1.2.3.4/32 -m comment --comment \"src:guid-%d\" -j MARK --set-xmark 0xa/0xffffffff", i)))
		}
	})

	It("supports concurrent bulk inserts and append uniques", func() {
		onlyRunOnLinux()
		numRules := 100
		numWorkers := 10
		things := []string{}
		for i := 0; i < numRules; i++ {
			things = append(things, fmt.Sprintf("%d", i))
		}
		runner := testsupport.ParallelRunner{
			NumWorkers: numWorkers,
		}
		restoreWorker := func(item string) {
			ruleItems := []rules.GenericRule{rules.NewMarkSetRule("1.3.5.7", "A", fmt.Sprintf("bulk-%s", item))}
			err := lockedIPT.BulkInsert("filter", "FORWARD", 1, ruleItems...)
			Expect(err).NotTo(HaveOccurred())

			r := rules.NewMarkSetRule("2.4.6.8", "A", fmt.Sprintf("uniq-%s", item))
			err = lockedIPT.AppendUnique("filter", "FORWARD", r.Properties...)
			Expect(err).NotTo(HaveOccurred())
		}
		runner.RunOnSliceStrings(things, restoreWorker)

		allRules := AllIPTablesRules("filter")
		for i := 0; i < numRules; i++ {
			Expect(allRules).To(ContainElement(
				fmt.Sprintf("-A FORWARD -s 1.3.5.7/32 -m comment --comment \"src:bulk-%d\" -j MARK --set-xmark 0xa/0xffffffff", i)))
			Expect(allRules).To(ContainElement(
				fmt.Sprintf("-A FORWARD -s 2.4.6.8/32 -m comment --comment \"src:uniq-%d\" -j MARK --set-xmark 0xa/0xffffffff", i)))
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
	Eventually(iptablesSession, "3s").Should(gexec.Exit(0))
	return strings.Split(strings.TrimSpace(string(iptablesSession.Out.Contents())), "\n")
}
