package integration_test

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"vxlan-policy-agent/config"

	"code.cloudfoundry.org/go-db-helpers/metrics"
	"code.cloudfoundry.org/go-db-helpers/mutualtls"

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
		conf             config.VxlanPolicyAgent
		configFilePath   string
		fakeMetron       metrics.FakeMetron
		mockPolicyServer ifrit.Process
		serverListenPort int
		serverListenAddr string
		serverTLSConfig  *tls.Config
	)

	BeforeEach(func() {
		var err error
		fakeMetron = metrics.NewFakeMetron()

		serverTLSConfig, err = mutualtls.NewServerTLSConfig(paths.ServerCertFile, paths.ServerKeyFile, paths.ClientCACertFile)
		Expect(err).NotTo(HaveOccurred())

		serverListenPort = 40000 + GinkgoParallelNode()
		serverListenAddr = fmt.Sprintf("127.0.0.1:%d", serverListenPort)

		containerMetadata := `
{
	"some-handle": {
		"handle":"some-handle",
		"ip":"10.255.100.21",
		"metadata": {
			"policy_group_id":"some-very-very-long-app-guid"
		}
	}
}
`
		containerMetadataFile, err := ioutil.TempFile("", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(ioutil.WriteFile(containerMetadataFile.Name(), []byte(containerMetadata), os.ModePerm))
		datastorePath = containerMetadataFile.Name()

		conf = config.VxlanPolicyAgent{
			PollInterval:         1,
			PolicyServerURL:      fmt.Sprintf("https://%s", serverListenAddr),
			Datastore:            datastorePath,
			VNI:                  42,
			MetronAddress:        fakeMetron.Address(),
			ServerCACertFile:     paths.ServerCACertFile,
			ClientCertFile:       paths.ClientCertFile,
			ClientKeyFile:        paths.ClientKeyFile,
			IPTablesLockFile:     GlobalIPTablesLockFile,
			DebugServerHost:      "127.0.0.1",
			DebugServerPort:      22222 + GinkgoParallelNode(),
			ClientTimeoutSeconds: 5,
		}
		Expect(conf.Validate()).To(Succeed())
		configFilePath = WriteConfigFile(conf)
	})

	AfterEach(func() {
		stopServer(mockPolicyServer)
		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())

		runIptablesCommand("filter", "F")
		runIptablesCommand("filter", "X")
		runIptablesCommand("nat", "F")
		runIptablesCommand("nat", "X")

		Expect(fakeMetron.Close()).To(Succeed())
	})

	setIPTablesLogging := func(enabled bool) {
		endpoint := fmt.Sprintf("http://%s:%d/iptables-c2c-logging", conf.DebugServerHost, conf.DebugServerPort)
		req, err := http.NewRequest("PUT", endpoint, strings.NewReader(fmt.Sprintf(`{ "enabled": %t }`, enabled)))
		Expect(err).NotTo(HaveOccurred())
		resp, err := http.DefaultClient.Do(req)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(ioutil.ReadAll(resp.Body)).To(MatchJSON(fmt.Sprintf(`{ "enabled": %t }`, enabled)))
	}

	Describe("policy agent", func() {
		BeforeEach(func() {
			mockPolicyServer = startServer(serverListenAddr, serverTLSConfig)
			session = startAgent(paths.VxlanPolicyAgentPath, configFilePath)
		})

		It("should boot and gracefully terminate", func() {
			Consistently(session).ShouldNot(gexec.Exit())
			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		})

		getIPTablesLogging := func() (bool, error) {
			endpoint := fmt.Sprintf("http://%s:%d/iptables-c2c-logging", conf.DebugServerHost, conf.DebugServerPort)
			resp, err := http.DefaultClient.Get(endpoint)
			if err != nil {
				return false, err
			}
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			var respStruct struct {
				Enabled bool `json:"enabled"`
			}
			Expect(json.NewDecoder(resp.Body).Decode(&respStruct)).To(Succeed())
			return respStruct.Enabled, nil
		}

		Describe("the debug server", func() {
			It("has a iptables logging endpoint", func() {
				Eventually(getIPTablesLogging).Should(BeFalse())
				setIPTablesLogging(LoggingEnabled)
				Expect(getIPTablesLogging()).To(BeTrue())
			})
		})

		It("supports enabling/disabling iptables logging at runtime", func() {
			By("checking that the logging rules are absent")
			Eventually(iptablesFilterRules, "4s", "0.5s").Should(MatchRegexp(PolicyRulesRegexp(LoggingDisabled)))

			By("enabling iptables logging")
			setIPTablesLogging(LoggingEnabled)

			By("checking that the logging rules are present")
			Eventually(iptablesFilterRules, "4s", "0.5s").Should(MatchRegexp(PolicyRulesRegexp(LoggingEnabled)))

			By("disabling iptables logging")
			setIPTablesLogging(LoggingDisabled)

			By("checking that the logging rules are absent")
			Eventually(iptablesFilterRules, "4s", "0.5s").Should(MatchRegexp(PolicyRulesRegexp(LoggingDisabled)))
		})

		It("writes the mark rule and enforces policies", func() {
			Eventually(iptablesFilterRules, "4s", "1s").Should(ContainSubstring(`-s 10.255.100.21/32 -m comment --comment "src:some-very-very-long-app-guid" -j MARK --set-xmark 0xa/0xffffffff`))
			Expect(iptablesFilterRules()).To(ContainSubstring(`-d 10.255.100.21/32 -p tcp -m tcp --dport 9999 -m mark --mark 0xc -m comment --comment "src:another-app-guid_dst:some-very-very-long-app-guid" -j ACCEPT`))
		})

		It("writes only one mark rule for a single container", func() {
			Eventually(iptablesFilterRules, "4s", "1s").Should(ContainSubstring(`-s 10.255.100.21/32 -m comment --comment "src:some-very-very-long-app-guid" -j MARK --set-xmark 0xa/0xffffffff`))
			Expect(iptablesFilterRules()).NotTo(MatchRegexp(`.*--set-xmark.*\n.*--set-xmark.*`))
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

		It("has a log level thats configurable at runtime", func() {
			Consistently(session).ShouldNot(gexec.Exit())
			Eventually(session.Out).Should(Say("container-networking.vxlan-policy-agent"))
			Consistently(session.Out).ShouldNot(Say("got-containers"))

			endpoint := fmt.Sprintf("http://%s:%d/log-level", conf.DebugServerHost, conf.DebugServerPort)
			req, err := http.NewRequest("POST", endpoint, strings.NewReader("debug"))
			Expect(err).NotTo(HaveOccurred())
			_, err = http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session.Out, "5s").Should(Say("container-networking.vxlan-policy-agent.*got-containers"))
		})
	})

	Context("when the vxlan policy agent cannot connect to the server upon start", func() {
		BeforeEach(func() {
			conf.PolicyServerURL = "some-bad-url"
			configFilePath = WriteConfigFile(conf)
			session = startAgent(paths.VxlanPolicyAgentPath, configFilePath)
		})

		It("crashes and logs a useful error message", func() {
			Eventually(session).Should(gexec.Exit())
			Eventually(session.Out.Contents).Should(MatchRegexp("policy-client-get-policies.*http client do.*unsupported protocol scheme"))
		})

	})

	Context("when the policy server is unavailable", func() {
		BeforeEach(func() {
			session = startAgent(paths.VxlanPolicyAgentPath, configFilePath)
		})

		It("does not write the mark rule or enforces policies", func() {
			Expect(iptablesFilterRules()).NotTo(ContainSubstring(`-s 10.255.100.21/32 -m comment --comment "src:some-very-very-long-app-guid" -j MARK --set-xmark 0xa/0xffffffff`))
			Expect(iptablesFilterRules()).NotTo(ContainSubstring(`-d 10.255.100.21/32 -p tcp -m tcp --dport 9999 -m mark --mark 0xc -m comment --comment "src:another-app-guid_dst:some-very-very-long-app-guid" -j ACCEPT`))
		})

		It("writes the mark rule or enforces policies when the policy server becomes available again", func() {
			mockPolicyServer = startServer(serverListenAddr, serverTLSConfig)
			Eventually(iptablesFilterRules, "10s", "1s").Should(ContainSubstring(`-s 10.255.100.21/32 -m comment --comment "src:some-very-very-long-app-guid" -j MARK --set-xmark 0xa/0xffffffff`))
			Expect(iptablesFilterRules()).To(ContainSubstring(`-d 10.255.100.21/32 -p tcp -m tcp --dport 9999 -m mark --mark 0xc -m comment --comment "src:another-app-guid_dst:some-very-very-long-app-guid" -j ACCEPT`))
		})
	})

	Context("when vxlan policy agent has invalid certs", func() {
		BeforeEach(func() {
			conf = config.VxlanPolicyAgent{
				Datastore:            datastorePath,
				PollInterval:         1,
				PolicyServerURL:      "",
				VNI:                  42,
				MetronAddress:        fakeMetron.Address(),
				ServerCACertFile:     paths.ServerCACertFile,
				ClientCertFile:       "totally",
				ClientKeyFile:        "not-cool",
				DebugServerHost:      "127.0.0.1",
				DebugServerPort:      22222 + GinkgoParallelNode(),
				ClientTimeoutSeconds: 5,
			}
			configFilePath = WriteConfigFile(conf)
		})

		It("does not start", func() {
			session = startAgent(paths.VxlanPolicyAgentPath, configFilePath)
			Eventually(session).Should(gexec.Exit(1))
			Eventually(session.Out).Should(Say("unable to load cert or key"))
		})
	})

	Context("when requests to the policy server time out", func() {
		BeforeEach(func() {
			conf.ClientTimeoutSeconds = 1
			configFilePath = WriteConfigFile(conf)
			mustSucceed("iptables", "-A", "INPUT", "-p", "tcp", "--dport", strconv.Itoa(serverListenPort), "-j", "DROP")
		})

		AfterEach(func() {
			mustSucceed("iptables", "-D", "INPUT", "-p", "tcp", "--dport", strconv.Itoa(serverListenPort), "-j", "DROP")
		})

		It("times out requests", func() {
			session = startAgent(paths.VxlanPolicyAgentPath, configFilePath)
			Eventually(session.Out.Contents, "3s").Should(MatchRegexp("policy-client-get-policies.*request canceled while waiting for connection.*Client.Timeout exceeded"))
			session.Kill()
		})
	})

	Context("when vxlan policy agent is deployed with iptables logging enabled", func() {
		BeforeEach(func() {
			conf = config.VxlanPolicyAgent{
				PollInterval:         1,
				PolicyServerURL:      fmt.Sprintf("https://%s", serverListenAddr),
				Datastore:            datastorePath,
				VNI:                  42,
				MetronAddress:        fakeMetron.Address(),
				ServerCACertFile:     paths.ServerCACertFile,
				ClientCertFile:       paths.ClientCertFile,
				ClientKeyFile:        paths.ClientKeyFile,
				IPTablesLockFile:     GlobalIPTablesLockFile,
				DebugServerHost:      "127.0.0.1",
				DebugServerPort:      22222 + GinkgoParallelNode(),
				IPTablesLogging:      true,
				ClientTimeoutSeconds: 5,
			}
			Expect(conf.Validate()).To(Succeed())
			configFilePath = WriteConfigFile(conf)
			mockPolicyServer = startServer(serverListenAddr, serverTLSConfig)
			session = startAgent(paths.VxlanPolicyAgentPath, configFilePath)
		})

		It("supports enabling/disabling iptables logging at runtime", func() {
			Consistently(session).ShouldNot(gexec.Exit())

			By("checking that the logging rules are present")
			Eventually(iptablesFilterRules, "2s", "0.5s").Should(MatchRegexp(PolicyRulesRegexp(LoggingEnabled)))

			By("disabling iptables logging")
			setIPTablesLogging(LoggingDisabled)

			By("checking that the logging rules are absent")
			Eventually(iptablesFilterRules, "2s", "0.5s").Should(MatchRegexp(PolicyRulesRegexp(LoggingDisabled)))

			By("enabling iptables logging")
			setIPTablesLogging(LoggingEnabled)

			By("checking that the logging rules are present")
			Eventually(iptablesFilterRules, "2s", "0.5s").Should(MatchRegexp(PolicyRulesRegexp(LoggingEnabled)))
		})
	})
})

func mustSucceed(binary string, args ...string) string {
	cmd := exec.Command(binary, args...)
	sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
	return string(sess.Out.Contents())
}

func iptablesFilterRules() string {
	return runIptablesCommand("filter", "S")
}

func iptablesNATRules() string {
	return runIptablesCommand("nat", "S")
}

func runIptablesCommand(table, flag string) string {
	iptCmd := exec.Command("iptables", "-w", "-t", table, "-"+flag)
	iptablesSession, err := gexec.Start(iptCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(iptablesSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
	return string(iptablesSession.Out.Contents())
}

func startAgent(binaryPath, configPath string) *gexec.Session {
	cmd := exec.Command(binaryPath, "-config-file", configPath)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return session
}

func startServer(serverListenAddr string, tlsConfig *tls.Config) ifrit.Process {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/networking/v0/internal/policies" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"policies": [
				{"source": {"id":"some-very-very-long-app-guid", "tag":"A"},
				"destination": {"id": "some-other-app-guid", "tag":"B", "protocol":"tcp", "port":3333}},
				{"source": {"id":"some-very-very-long-app-guid", "tag":"A"},
				"destination": {"id": "some-other-app-guid", "tag":"B", "protocol":"tcp", "port":3334}},
				{"source": {"id":"another-app-guid", "tag":"C"},
				"destination": {"id": "some-very-very-long-app-guid", "tag":"A", "protocol":"tcp", "port":9999}}
					]}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
		return
	})
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

func stopServer(server ifrit.Process) {
	if server == nil {
		return
	}
	server.Signal(os.Interrupt)
	Eventually(server.Wait()).Should(Receive())
}

const (
	LoggingDisabled = false
	LoggingEnabled  = true
)

func PolicyRulesRegexp(loggingEnabled bool) string {
	policyRules := ""
	if loggingEnabled {
		policyRules += `.*-A vpa--[0-9]+ -d 10.255.100.21/32 -p tcp -m tcp --dport 9999 -m mark --mark 0xc -m limit --limit 2/min -j LOG --log-prefix "OK_C_some-very-very-long-app "\n`
	}
	policyRules += `.*-A vpa--[0-9]+ -d 10.255.100.21/32 -p tcp -m tcp --dport 9999 -m mark --mark 0xc -m comment --comment "src:another-app-guid_dst:some-very-very-long-app-guid" -j ACCEPT`
	return policyRules
}
