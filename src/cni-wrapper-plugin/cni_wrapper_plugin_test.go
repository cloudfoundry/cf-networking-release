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
		cmd             *exec.Cmd
		debugFileName   string
		datastorePath   string
		input           string
		debug           *noop_debug.Debug
		expectedCmdArgs skel.CmdArgs
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
		}
		toReturn.Stdin = strings.NewReader(input)
		return toReturn
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
		input = fmt.Sprintf(inputTemplate, datastorePath)

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
	})

	Describe("state lifecylcle", func() {
		It("stores and removes metadata with the lifetime of the container", func() {
			By("calling ADD")
			cmd.Env[0] = "CNI_COMMAND=ADD"
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			By("check that metadata is stored")
			stateFileBytes, err := ioutil.ReadFile(datastorePath)
			Expect(err).NotTo(HaveOccurred())
			fmt.Println(datastorePath)
			Expect(string(stateFileBytes)).To(ContainSubstring(fmt.Sprintf("1.2.3.4")))

			By("calling DEL")
			cmd = cniCommand("DEL", input)
			cmd.Env[0] = "CNI_COMMAND=DEL"
			session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			By("check that metadata is has been removed")
			stateFileBytes, err = ioutil.ReadFile(datastorePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stateFileBytes)).NotTo(ContainSubstring(fmt.Sprintf("1.2.3.4")))
		})
	})

	Context("When call with command ADD", func() {
		BeforeEach(func() {
			cmd.Env[0] = "CNI_COMMAND=ADD"
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

		Context("When the delegate plugin return an error", func() {
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

			It("wraps and returns the error", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				Expect(session.Out.Contents()).To(MatchJSON(`{ "code": 100, "msg": "delegate call: banana" }`))
			})
		})

	})
})
