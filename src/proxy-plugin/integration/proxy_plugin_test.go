package main_test

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/containernetworking/plugins/pkg/ns"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"os"
	"path/filepath"
	"proxy-plugin/lib"
)

type InputStruct struct {
	Name       string `json:"name"`
	CNIVersion string `json:"cniVersion"`
	Type       string `json:"type"`
	lib.ProxyConfig
}

var _ = Describe("CniWrapperPlugin", func() {
	var (
		cmd                  *exec.Cmd
		debugFileName        string
		input                string
		inputStruct          InputStruct
		containerID          string
		proxyChainName       string
		containerNetNS       ns.NetNS
		containerNSShortName string
	)

	var cniCommand = func(command, input string) *exec.Cmd {
		if _, err := os.Stat(paths.PathToPlugin); os.IsNotExist(err) {
			Expect(err).ToNot(HaveOccurred())
		}

		toReturn := exec.Command(paths.PathToPlugin)
		toReturn.Env = []string{
			"CNI_COMMAND=" + command,
			"CNI_CONTAINERID=" + containerID,
			"CNI_NETNS=" + containerNetNS.Path(),
			"CNI_IFNAME=some-eth0",
			"CNI_PATH=" + paths.CNIPath,
			"CNI_ARGS=DEBUG=" + debugFileName,
			"PATH=/sbin",
		}
		toReturn.Stdin = strings.NewReader(input)

		return toReturn
	}

	ContainerIPTablesRules := func(containerNetns string, tableName string) []string {
		iptablesSession, err := gexec.Start(exec.Command("ip", "netns", "exec", containerNetns, "iptables", "-w", "-S", "-t", tableName), GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(iptablesSession).Should(gexec.Exit(0))
		return strings.Split(string(iptablesSession.Out.Contents()), "\n")
	}

	GetInput := func(i InputStruct) string {
		inputBytes, err := json.Marshal(i)
		Expect(err).NotTo(HaveOccurred())
		return string(inputBytes)
	}

	BeforeEach(func() {
		containerNetNS = createNetworkNamespace()
		containerNSShortName = filepath.Base(containerNetNS.Path())

		inputStruct = InputStruct{
			Name:       "proxy-plugin",
			CNIVersion: "0.3.1",
			Type:       "proxy-plugin",
			ProxyConfig: lib.ProxyConfig{
				ProxyRange: "10.255.0.0/16",
				ProxyPort:  8675, //randomize for parallel execution
			},
		}

		input = GetInput(inputStruct)

		containerID = "some-container-id-that-is-long" //randomize for parallel execution

		proxyChainName = ("proxy--" + containerID)[:28]
		cmd = cniCommand("ADD", input)

		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Eventually(session).Should(gexec.Exit(0))
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		removeNetworkNamespace(containerNetNS)
	})

	Context("When call with command ADD", func() {
		It("writes redirect output chain rules to proxy traffic envoy in the container namespace", func() {
			By("checking that the envoy chain is created")
			Expect(ContainerIPTablesRules(containerNSShortName, "nat")).To(ContainElement("-N " + proxyChainName))

			By("checking that the output chain jumps to the envoy chain")
			Expect(ContainerIPTablesRules(containerNSShortName, "nat")).To(ContainElement("-A OUTPUT -j " + proxyChainName))

			By("checking that the envoy chain returns when the owner is not vcap")
			Expect(ContainerIPTablesRules(containerNSShortName, "nat")).To(ContainElement(fmt.Sprintf("-A %s -m owner ! --uid-owner 1000 -j RETURN", proxyChainName)))

			By("checking that the envoy chain redirects to the proxy port")
			Expect(ContainerIPTablesRules(containerNSShortName, "nat")).To(ContainElement(fmt.Sprintf("-A %s -d %s -p tcp -j REDIRECT --to-ports %d", proxyChainName, inputStruct.ProxyRange, inputStruct.ProxyPort)))
		})
	})

	Context("When call with command DEL", func() {
		It("runs without explosion", func() {
			cmd.Env[0] = "CNI_COMMAND=DEL"
			cmd := cniCommand("DEL", input)
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

			Eventually(session, "5s").Should(gexec.Exit(0))
			Expect(err).NotTo(HaveOccurred())

			By("checking that the envoy chain is not present")
			Expect(ContainerIPTablesRules(containerNSShortName, "nat")).ToNot(ContainElement("-N " + proxyChainName))
		})
	})
})

func createNetworkNamespace() ns.NetNS {
	networkNS, err := ns.NewNS()
	Expect(err).ToNot(HaveOccurred())
	return networkNS
}

func removeNetworkNamespace(containerNetNs ns.NetNS) {
	Expect(containerNetNs.Close()).To(Succeed())
}
