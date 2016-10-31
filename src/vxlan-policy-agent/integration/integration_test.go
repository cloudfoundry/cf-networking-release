package integration_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"lib/mutualtls"
	"net/http"
	"netmon/integration/fakes"
	"os"
	"os/exec"
	"vxlan-policy-agent/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

var _ = Describe("VXLAN Policy Agent", func() {
	var (
		session          *gexec.Session
		datastorePath    string
		subnetFile       *os.File
		configFilePath   string
		fakeMetron       fakes.FakeMetron
		mockPolicyServer ifrit.Process

		serverListenAddr string
	)

	startServer := func(tlsConfig *tls.Config) ifrit.Process {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/networking/v0/internal/policies" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"policies": [
				{"source": {"id":"some-app-guid", "tag":"A"},
				"destination": {"id": "some-other-app-guid", "tag":"B", "protocol":"tcp", "port":3333}},
				{"source": {"id":"another-app-guid", "tag":"C"},
				"destination": {"id": "some-app-guid", "tag":"A", "protocol":"tcp", "port":9999}}
					]}`))
				return
			}

			w.WriteHeader(http.StatusNotFound)
			return
		})
		serverListenAddr = fmt.Sprintf("127.0.0.1:%d", 40000+GinkgoParallelNode())
		someServer := http_server.NewTLSServer(serverListenAddr, testHandler, tlsConfig)

		members := grouper.Members{{
			Name:   "http_server",
			Runner: someServer,
		}}
		group := grouper.NewOrdered(os.Interrupt, members)
		monitor := ifrit.Invoke(sigmon.New(group))

		Eventually(monitor.Ready()).Should(BeClosed())
		return monitor
	}

	BeforeEach(func() {
		var err error
		fakeMetron = fakes.New()

		serverTLSConfig, err := mutualtls.NewServerTLSConfig(paths.ServerCertFile, paths.ServerKeyFile, paths.ClientCACertFile)
		Expect(err).NotTo(HaveOccurred())

		mockPolicyServer = startServer(serverTLSConfig)

		subnetFile, err = ioutil.TempFile("", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(ioutil.WriteFile(subnetFile.Name(), []byte("FLANNEL_NETWORK=10.255.0.0/16\nFLANNEL_SUBNET=10.255.100.1/24"), os.ModePerm))

		containerMetadata := `
{
	"some-handle": {
		"handle":"some-handle",
		"ip":"10.255.100.21",
		"metadata": {
			"policy_group_id":"some-app-guid"
		}
	}
}
`
		containerMetadataFile, err := ioutil.TempFile("", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(ioutil.WriteFile(containerMetadataFile.Name(), []byte(containerMetadata), os.ModePerm))
		datastorePath = containerMetadataFile.Name()

		conf := config.VxlanPolicyAgent{
			PollInterval:      1,
			PolicyServerURL:   fmt.Sprintf("https://%s", serverListenAddr),
			Datastore:         datastorePath,
			VNI:               42,
			FlannelSubnetFile: subnetFile.Name(),
			MetronAddress:     fakeMetron.Address(),
			ServerCACertFile:  paths.ServerCACertFile,
			ClientCertFile:    paths.ClientCertFile,
			ClientKeyFile:     paths.ClientKeyFile,
		}
		configFilePath = WriteConfigFile(conf)
	})

	AfterEach(func() {
		mockPolicyServer.Signal(os.Interrupt)
		Eventually(mockPolicyServer.Wait()).Should(Receive())

		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())

		RunIptablesCommand("filter", "F")
		RunIptablesCommand("filter", "X")
		RunIptablesCommand("nat", "F")
		RunIptablesCommand("nat", "X")

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Describe("boring daemon behavior", func() {
		It("should boot and gracefully terminate", func() {
			session = StartAgent(paths.VxlanPolicyAgentPath, configFilePath)
			Consistently(session).ShouldNot(gexec.Exit())

			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		})
	})

	var waitUntilPollLoop = func(numComplete int) {
		pollLoopCount := func() int {
			count := 0
			events := fakeMetron.AllEvents()
			for _, event := range events {
				if event.Name == "containerMetadataTime" {
					count++
				}
			}
			return count
		}
		Eventually(pollLoopCount, DEFAULT_TIMEOUT).Should(BeNumerically(">=", numComplete+1))
	}

	Describe("Default rules", func() {
		BeforeEach(func() {
			session = StartAgent(paths.VxlanPolicyAgentPath, configFilePath)
		})

		It("writes the masquerade rule", func() {
			waitUntilPollLoop(1)

			ipTablesRules := RunIptablesCommand("nat", "S")

			Expect(ipTablesRules).To(ContainSubstring("-s 10.255.100.0/24 ! -d 10.255.0.0/16 -j MASQUERADE"))
		})

		It("writes the default remote rules", func() {
			waitUntilPollLoop(1)

			ipTablesRules := RunIptablesCommand("filter", "S")

			Expect(ipTablesRules).To(ContainSubstring("-i flannel.42 -m state --state RELATED,ESTABLISHED -j ACCEPT"))
			Expect(ipTablesRules).To(ContainSubstring("-i flannel.42 -j REJECT --reject-with icmp-port-unreachable"))
		})

		It("writes the default local rules", func() {
			waitUntilPollLoop(1)

			ipTablesRules := RunIptablesCommand("filter", "S")

			Expect(ipTablesRules).To(ContainSubstring("-i cni-flannel0 -m state --state RELATED,ESTABLISHED -j ACCEPT"))
			Expect(ipTablesRules).To(ContainSubstring("-s 10.255.100.0/24 -d 10.255.100.0/24 -i cni-flannel0 -j REJECT --reject-with icmp-port-unreachable"))
		})
	})

	Describe("policy enforcement", func() {
		BeforeEach(func() {
			session = StartAgent(paths.VxlanPolicyAgentPath, configFilePath)
		})
		It("writes the mark rule", func() {
			waitUntilPollLoop(2) // wait for a second one so we know the first enforcement completed

			ipTablesRules := RunIptablesCommand("filter", "S")

			Expect(ipTablesRules).To(ContainSubstring(`-s 10.255.100.21/32 -m comment --comment "src:some-app-guid" -j MARK --set-xmark 0xa/0xffffffff`))
		})
		It("enforces policies", func() {
			waitUntilPollLoop(2) // wait for a second one so we know the first enforcement completed

			ipTablesRules := RunIptablesCommand("filter", "S")

			Expect(ipTablesRules).To(ContainSubstring(`-d 10.255.100.21/32 -p tcp -m tcp --dport 9999 -m mark --mark 0xc -m comment --comment "src:another-app-guid dst:some-app-guid" -j ACCEPT`))
		})
	})

	Describe("reporting metrics", func() {
		BeforeEach(func() {
			session = StartAgent(paths.VxlanPolicyAgentPath, configFilePath)
		})
		It("emits metrics about durations", func() {
			gatherMetricNames := func() map[string]bool {
				events := fakeMetron.AllEvents()
				metrics := map[string]bool{}
				for _, event := range events {
					metrics[event.Name] = true
				}
				return metrics
			}
			Eventually(gatherMetricNames, "5s").Should(HaveKey("iptablesEnforceTime"))
			Eventually(gatherMetricNames, "5s").Should(HaveKey("totalPollTime"))
			Eventually(gatherMetricNames, "5s").Should(HaveKey("containerMetadataTime"))
			Eventually(gatherMetricNames, "5s").Should(HaveKey("policyServerPollTime"))
		})
	})

	Context("when the policy server is unavailable", func() {
		BeforeEach(func() {
			conf := config.VxlanPolicyAgent{
				Datastore:         datastorePath,
				PollInterval:      1,
				PolicyServerURL:   "",
				VNI:               42,
				FlannelSubnetFile: subnetFile.Name(),
				MetronAddress:     fakeMetron.Address(),
				ServerCACertFile:  paths.ServerCACertFile,
				ClientCertFile:    paths.ClientCertFile,
				ClientKeyFile:     paths.ClientKeyFile,
			}
			configFilePath = WriteConfigFile(conf)
			session = StartAgent(paths.VxlanPolicyAgentPath, configFilePath)
		})

		It("still writes the default rules", func() {
			waitUntilPollLoop(2) // wait for a second one so we know the first enforcement completed

			ipTablesRules := RunIptablesCommand("filter", "S")
			Expect(ipTablesRules).To(ContainSubstring("-i flannel.42 -m state --state RELATED,ESTABLISHED -j ACCEPT"))
			Expect(ipTablesRules).To(ContainSubstring("-i flannel.42 -j REJECT --reject-with icmp-port-unreachable"))
			Expect(ipTablesRules).To(ContainSubstring("-i cni-flannel0 -m state --state RELATED,ESTABLISHED -j ACCEPT"))
			Expect(ipTablesRules).To(ContainSubstring("-s 10.255.100.0/24 -d 10.255.100.0/24 -i cni-flannel0 -j REJECT --reject-with icmp-port-unreachable"))

			ipTablesRules = RunIptablesCommand("nat", "S")
			Expect(ipTablesRules).To(ContainSubstring("-s 10.255.100.0/24 ! -d 10.255.0.0/16 -j MASQUERADE"))
		})
	})

	Context("when vxlan policy agent has invalid certs", func() {
		BeforeEach(func() {
			conf := config.VxlanPolicyAgent{
				Datastore:         datastorePath,
				PollInterval:      1,
				PolicyServerURL:   "",
				VNI:               42,
				FlannelSubnetFile: subnetFile.Name(),
				MetronAddress:     fakeMetron.Address(),
				ServerCACertFile:  paths.ServerCACertFile,
				ClientCertFile:    "totally",
				ClientKeyFile:     "not-cool",
			}
			configFilePath = WriteConfigFile(conf)
		})

		It("does not start", func() {
			session = StartAgent(paths.VxlanPolicyAgentPath, configFilePath)
			Eventually(session).Should(gexec.Exit(1))
			Eventually(session.Out).Should(Say("unable to load cert or key"))
		})
	})
})

func RunIptablesCommand(table, flag string) string {
	iptCmd := exec.Command("iptables", "-w", "-t", table, "-"+flag)
	iptablesSession, err := gexec.Start(iptCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(iptablesSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
	return string(iptablesSession.Out.Contents())
}

func StartAgent(binaryPath, configPath string) *gexec.Session {
	cmd := exec.Command(binaryPath, "-config-file", configPath)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return session
}
