package main_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/containernetworking/cni/pkg/skel"
	noop_debug "github.com/containernetworking/cni/plugins/test/noop/debug"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("CniWrapperPlugin", func() {

	var (
		cmd                  *exec.Cmd
		debugFileName        string
		datastorePath        string
		iptablesLockFilePath string
		input                string
		debug                *noop_debug.Debug
		expectedCmdArgs      skel.CmdArgs
	)

	const delegateInput = `
{
		"type": "noop",
		"some": "other data"
}
`

	const inputTemplate = `
{
  "name": "cni-wrapper",
  "type": "wrapper",
  "datastore": "%s",
  "iptables_lock_file": "%s",
  "overlay_network": "%s",

	"metadata": {
			"key1": "value1",
			"key2": [ "some", "data" ]
	},

	"delegate": ` +
		delegateInput +
		`}`

	var cniCommand = func(command, input string) *exec.Cmd {
		toReturn := exec.Command(paths.PathToPlugin)
		toReturn.Env = []string{
			"CNI_COMMAND=" + command,
			"CNI_CONTAINERID=some-container-id",
			"CNI_NETNS=/some/netns/path",
			"CNI_IFNAME=some-eth0",
			"CNI_PATH=" + paths.CNIPath,
			"CNI_ARGS=DEBUG=" + debugFileName,
			"PATH=/sbin",
		}
		toReturn.Stdin = strings.NewReader(input)

		return toReturn
	}

	var iptablesNATRules = func() string {
		iptCmd := exec.Command("iptables", "-w", "-t", "nat", "-S")
		iptablesSession, err := gexec.Start(iptCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(iptablesSession).Should(gexec.Exit(0))
		return string(iptablesSession.Out.Contents())
	}

	BeforeEach(func() {
		debugFile, err := ioutil.TempFile("", "cni_debug")
		Expect(err).NotTo(HaveOccurred())
		Expect(debugFile.Close()).To(Succeed())
		debugFileName = debugFile.Name()

		debug = &noop_debug.Debug{
			ReportResult:         `{ "ip4": { "ip": "1.2.3.4/32" } }`,
			ReportVersionSupport: []string{"0.1.0", "0.2.0", "0.3.0"},
		}
		Expect(debug.WriteDebug(debugFileName)).To(Succeed())

		datastoreFile, err := ioutil.TempFile("", "datastore")
		Expect(err).NotTo(HaveOccurred())
		Expect(datastoreFile.Close()).To(Succeed())
		datastorePath = datastoreFile.Name()

		iptablesLockFile, err := ioutil.TempFile("", "iptables-lock")
		Expect(err).NotTo(HaveOccurred())
		Expect(iptablesLockFile.Close()).To(Succeed())
		iptablesLockFilePath = iptablesLockFile.Name()

		input = fmt.Sprintf(inputTemplate, datastorePath, iptablesLockFilePath, "10.255.0.0/16")

		expectedCmdArgs = skel.CmdArgs{
			ContainerID: "some-container-id",
			Netns:       "/some/netns/path",
			IfName:      "some-eth0",
			Args:        "DEBUG=" + debugFileName,
			Path:        "/some/bin/path",
			StdinData:   []byte(input),
		}
		cmd = cniCommand("ADD", input)
	})

	AfterEach(func() {
		os.Remove(debugFileName)
		os.Remove(datastorePath)
		os.Remove(iptablesLockFilePath)
	})

	Describe("state lifecycle", func() {
		It("stores and removes metadata with the lifetime of the container", func() {
			By("calling ADD")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			By("check that metadata is stored")
			stateFileBytes, err := ioutil.ReadFile(datastorePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stateFileBytes)).To(ContainSubstring("1.2.3.4"))
			Expect(string(stateFileBytes)).To(ContainSubstring("value1"))

			By("calling DEL")
			cmd = cniCommand("DEL", input)
			session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			By("check that metadata is has been removed")
			stateFileBytes, err = ioutil.ReadFile(datastorePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stateFileBytes)).NotTo(ContainSubstring("1.2.3.4"))
			Expect(string(stateFileBytes)).NotTo(ContainSubstring("value1"))
		})
	})

	Describe("iptables lifecycle", func() {
		It("adds and removes ip masquerade rules with the lifetime of the container", func() {
			By("calling ADD")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			By("check that ip masquerade rule is created")
			Expect(iptablesNATRules()).To(ContainSubstring("-A POSTROUTING -s 1.2.3.4/32 ! -d 10.255.0.0/16 -j MASQUERADE"))

			By("calling DEL")
			cmd = cniCommand("DEL", input)
			session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			By("check that ip masquerade rule is removed")
			Expect(iptablesNATRules()).NotTo(ContainSubstring("-A POSTROUTING -s 1.2.3.4/32 ! -d 10.255.0.0/16 -j MASQUERADE"))
		})
	})

	Context("When call with command ADD", func() {
		AfterEach(func() {
			cmd := cniCommand("DEL", input)
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			Expect(iptablesNATRules()).NotTo(ContainSubstring("-A POSTROUTING -s 1.2.3.4/32 ! -d 10.255.0.0/16 -j MASQUERADE"))
		})

		It("passes the delegate result back to the caller", func() {
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(MatchJSON(`{ "ip4": { "ip": "1.2.3.4/32" }, "dns":{} }`))
		})

		It("passes the correct stdin to the delegate plugin", func() {
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			debug, err := noop_debug.ReadDebug(debugFileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(debug.Command).To(Equal("ADD"))

			Expect(debug.CmdArgs.StdinData).To(MatchJSON(delegateInput))
		})

		It("ensures the container masquerade rule is created", func() {
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(MatchJSON(`{ "ip4": { "ip": "1.2.3.4/32" }, "dns":{} }`))
			Expect(iptablesNATRules()).To(ContainSubstring("-A POSTROUTING -s 1.2.3.4/32 ! -d 10.255.0.0/16 -j MASQUERADE"))
		})

		Context("When the delegate plugin returns an error", func() {
			BeforeEach(func() {
				debug.ReportError = "banana"
				Expect(debug.WriteDebug(debugFileName)).To(Succeed())
			})

			It("wraps and returns the error", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				Expect(session.Out.Contents()).To(MatchJSON(`{ "code": 100, "msg": "delegate call: banana" }`))
			})
		})

		Context("when the datastore add fails", func() {
			BeforeEach(func() {
				cmd.Env[1] = "CNI_CONTAINERID="
			})

			It("wraps and returns the error", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				Expect(session.Out.Contents()).To(MatchJSON(`{ "code": 100, "msg": "store add: invalid handle" }`))
			})

			It("does not leave any iptables rules behind", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				Expect(iptablesNATRules()).NotTo(ContainSubstring("-A POSTROUTING -s 1.2.3.4/32 ! -d 10.255.0.0/16 -j MASQUERADE"))
			})
		})

		Context("when the CNI call has no metadata", func() {
			BeforeEach(func() {
				inputTemplate := `
{
  "name": "cni-wrapper",
  "type": "wrapper",
  "datastore": "%s",
	"iptables_lock_file": "%s",
  "overlay_network": "%s",
	"delegate": ` +
					delegateInput +
					`}`
				input = fmt.Sprintf(inputTemplate, datastorePath, iptablesLockFilePath, "10.255.0.0/16")
			})
			It("succeeds and writes container IP to the datastore", func() {
				cmd = cniCommand("ADD", input)
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("check that metadata is stored")
				stateFileBytes, err := ioutil.ReadFile(datastorePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(stateFileBytes)).To(ContainSubstring("1.2.3.4"))

				By("calling DEL")
				cmd = cniCommand("DEL", input)
				session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("check that metadata is has been removed")
				stateFileBytes, err = ioutil.ReadFile(datastorePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(stateFileBytes)).NotTo(ContainSubstring("1.2.3.4"))
			})
		})
	})

	Context("When call with command DEL", func() {
		BeforeEach(func() {
			cmd.Env[0] = "CNI_COMMAND=DEL"
		})

		It("passes the correct stdin to the delegate plugin", func() {
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			debug, err := noop_debug.ReadDebug(debugFileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(debug.Command).To(Equal("DEL"))

			Expect(debug.CmdArgs.StdinData).To(MatchJSON(delegateInput))
		})

		Context("When the delegate plugin return an error", func() {
			BeforeEach(func() {
				debug.ReportError = "banana"
				Expect(debug.WriteDebug(debugFileName)).To(Succeed())
			})

			It("logs the wrapped error to stderr and return the success status code (for idempotency)", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				Expect(session.Err.Contents()).To(ContainSubstring("delegate delete: banana"))
			})
		})

		Context("when the datastore delete fails", func() {
			BeforeEach(func() {
				cmd.Env[1] = "CNI_CONTAINERID="
			})

			It("wraps and logs the error, and returns the success status code (for idempotency)", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				Expect(session.Err.Contents()).To(ContainSubstring("store delete: invalid handle"))
			})

			It("still calls plugin delete (so that DEL is idempotent)", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				debug, err := noop_debug.ReadDebug(debugFileName)
				Expect(err).NotTo(HaveOccurred())
				Expect(debug.Command).To(Equal("DEL"))

				Expect(debug.CmdArgs.StdinData).To(MatchJSON(delegateInput))
			})
		})

	})
})
