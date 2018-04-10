package lib_test

import (
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"lib/rules"
	"proxy-plugin/lib"
	"proxy-plugin/lib/fakes"
)

var _ = Describe("ContainerNSIPTables", func() {
	var (
		commandRunner       *fakes.CommandRunner
		containerNSIPTables lib.ContainerNSIPTables
	)
	BeforeEach(func() {
		commandRunner = &fakes.CommandRunner{}
		containerNSIPTables = lib.ContainerNSIPTables{
			CommandRunner:      commandRunner,
			ContainerNameSpace: "delightfuldweeb",
		}
	})

	Describe("New Chain", func() {
		It("executes the command line command with proper args", func() {
			err := containerNSIPTables.NewChain("table-beep", "chain-glamour")
			Expect(err).ToNot(HaveOccurred())

			Expect(commandRunner.ExecCallCount()).To(Equal(1))
			command, args := commandRunner.ExecArgsForCall(0)
			Expect(command).To(Equal("ip"))
			Expect(args).To(Equal([]string{"netns", "exec", "delightfuldweeb", "iptables", "-t", "table-beep", "-N", "chain-glamour"}))
		})

		Describe("when creation chain fails", func() {
			It("returns an error", func() {
				commandRunner.ExecReturns([]byte("bad args"), errors.New("exec failed"))

				err := containerNSIPTables.NewChain("table-beep", "chain-glamour")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(`new chain: failed running 'ip' with args: [netns exec delightfuldweeb iptables -t table-beep -N chain-glamour] output: "bad args" err: "exec failed"`))
			})
		})
	})

	Describe("BulkAppend", func() {
		It("executes the command line command with proper args", func() {
			err := containerNSIPTables.BulkAppend("thetable", "thechain",
				rules.IPTablesRule{"OUTPUT", "-j", "chain-name"},
				rules.IPTablesRule{"chain-name", "-m", "owner", "!", "--uid-owner", "1000", "-j", "RETURN"},
			)
			Expect(err).ToNot(HaveOccurred())

			Expect(commandRunner.ExecCallCount()).To(Equal(2))
			command, args := commandRunner.ExecArgsForCall(0)
			Expect(command).To(Equal("ip"))
			Expect(args).To(Equal([]string{"netns", "exec", "delightfuldweeb", "iptables", "-t", "thetable", "-A", "OUTPUT", "-j", "chain-name"}))

			command, args = commandRunner.ExecArgsForCall(1)
			Expect(command).To(Equal("ip"))
			Expect(args).To(Equal([]string{"netns", "exec", "delightfuldweeb", "iptables", "-t", "thetable", "-A", "chain-name", "-m", "owner", "!", "--uid-owner", "1000", "-j", "RETURN"}))
		})

		Describe("when one of the rules causes an error", func() {
			BeforeEach(func() {
				var callCount = 0
				commandRunner.ExecStub = func(name string, arg ...string) ([]byte, error) {
					if callCount > 0 {
						return []byte("bad args"), errors.New("execing command")
					}
					callCount++
					return []byte{}, nil
				}
			})

			It("returns the error", func() {
				err := containerNSIPTables.BulkAppend("thetable", "thechain",
					rules.IPTablesRule{"OUTPUT", "-j", "chain-name"},
					rules.IPTablesRule{"chain-name", "-m", "owner", "!", "--uid-owner", "1000", "-j", "RETURN"},
				)
				Expect(err).To(MatchError(`bulk append: failed running 'ip' with args: [netns exec delightfuldweeb iptables -t thetable -A chain-name -m owner ! --uid-owner 1000 -j RETURN] output: "bad args" err: "execing command"`))
			})
		})
	})

	Describe("DeleteChain", func() {
		It("executes the command line command with proper args", func() {
			err := containerNSIPTables.DeleteChain("thetable", "thechain")
			Expect(err).ToNot(HaveOccurred())

			Expect(commandRunner.ExecCallCount()).To(Equal(1))
			command, args := commandRunner.ExecArgsForCall(0)
			Expect(command).To(Equal("ip"))
			Expect(args).To(Equal([]string{"netns", "exec", "delightfuldweeb", "iptables", "-t", "thetable", "-X", "thechain"}))
		})

		Context("when deleting a chain fails", func() {
			BeforeEach(func() {
				commandRunner.ExecReturns([]byte("meow"), errors.New("floofy-pluto"))
			})

			It("returns an error", func() {
				err := containerNSIPTables.DeleteChain("thetable", "thechain")
				Expect(err).To(MatchError(`delete chain: failed running 'ip' with args: [netns exec delightfuldweeb iptables -t thetable -X thechain] output: "meow" err: "floofy-pluto"`))
			})
		})
	})

	Describe("Delete", func() {
		It("executes the command line command with proper args", func() {
			err := containerNSIPTables.Delete("thetable", "thechain",
				rules.IPTablesRule{"OUTPUT", "-j", "thechain"},
			)
			Expect(err).ToNot(HaveOccurred())

			Expect(commandRunner.ExecCallCount()).To(Equal(1))
			command, args := commandRunner.ExecArgsForCall(0)
			Expect(command).To(Equal("ip"))
			Expect(args).To(Equal([]string{"netns", "exec", "delightfuldweeb", "iptables", "-t", "thetable", "-D", "OUTPUT", "-j", "thechain"}))
		})

		Context("when deleting a chain fails", func() {
			BeforeEach(func() {
				commandRunner.ExecReturns([]byte("meow"), errors.New("floofy-pluto"))
			})

			It("returns an error", func() {
				err := containerNSIPTables.Delete("thetable", "thechain",
					rules.IPTablesRule{"OUTPUT", "-j", "thechain"},
				)
				Expect(err).To(MatchError(`delete: failed running 'ip' with args: [netns exec delightfuldweeb iptables -t thetable -D OUTPUT -j thechain] output: "meow" err: "floofy-pluto"`))
			})
		})
	})
})
