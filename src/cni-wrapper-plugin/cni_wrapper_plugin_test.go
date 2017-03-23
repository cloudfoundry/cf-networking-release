package main_test

import (
	"cni-wrapper-plugin/lib"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"

	"code.cloudfoundry.org/garden"

	noop_debug "github.com/containernetworking/cni/plugins/test/noop/debug"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/gomegamatchers"
)

type InputStruct struct {
	Name       string                 `json:"name"`
	CNIVersion string                 `json:"cniVersion"`
	Type       string                 `json:"type"`
	Delegate   map[string]interface{} `json:"delegate"`
	Metadata   map[string]interface{} `json:"metadata"`
	lib.WrapperConfig
}

var _ = Describe("CniWrapperPlugin", func() {

	var (
		cmd                     *exec.Cmd
		debugFileName           string
		datastorePath           string
		iptablesLockFilePath    string
		input                   string
		debug                   *noop_debug.Debug
		healthCheckServer       *httptest.Server
		healthCheckReturnStatus int
		inputStruct             InputStruct
		containerID             string
		netinChainName          string
		netoutChainName         string
		inputChainName          string
		netoutLoggingChainName  string
	)

	var cniCommand = func(command, input string) *exec.Cmd {
		toReturn := exec.Command(paths.PathToPlugin)
		toReturn.Env = []string{
			"CNI_COMMAND=" + command,
			"CNI_CONTAINERID=" + containerID,
			"CNI_NETNS=/some/netns/path",
			"CNI_IFNAME=some-eth0",
			"CNI_PATH=" + paths.CNIPath,
			"CNI_ARGS=DEBUG=" + debugFileName,
			"PATH=/sbin",
		}
		toReturn.Stdin = strings.NewReader(input)

		return toReturn
	}

	AllIPTablesRules := func(tableName string) []string {
		iptablesSession, err := gexec.Start(exec.Command("iptables", "-w", "-S", "-t", tableName), GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(iptablesSession).Should(gexec.Exit(0))
		return strings.Split(strings.TrimSpace(string(iptablesSession.Out.Contents())), "\n")
	}

	GetInput := func(i InputStruct) string {
		inputBytes, err := json.Marshal(i)
		Expect(err).NotTo(HaveOccurred())
		return string(inputBytes)
	}

	BeforeEach(func() {
		healthCheckReturnStatus = http.StatusOK
		healthCheckServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(healthCheckReturnStatus)
		}))

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

		inputStruct = InputStruct{
			Name:       "cni-wrapper",
			CNIVersion: "0.3.0",
			Type:       "wrapper",
			Delegate: map[string]interface{}{
				"type": "noop",
				"some": "other data",
			},
			Metadata: map[string]interface{}{
				"key1": "value1",
				"key2": []string{"some", "data"},
			},
			WrapperConfig: lib.WrapperConfig{
				Datastore:        datastorePath,
				HealthCheckURL:   healthCheckServer.URL,
				IPTablesLockFile: iptablesLockFilePath,
				OverlayNetwork:   "10.255.0.0/16",
				Delegate: map[string]interface{}{
					"type": "noop",
					"some": "other data",
				},
				InstanceAddress:    "10.244.2.3",
				IPTablesASGLogging: false,
				RuntimeConfig: &lib.RuntimeConfig{
					PortMappings: []garden.NetIn{
						{
							HostPort:      1000,
							ContainerPort: 1001,
						},
						{
							HostPort:      2000,
							ContainerPort: 2001,
						},
					},
					NetOutRules: []garden.NetOutRule{
						{
							Protocol: 1,
							Networks: []garden.IPRange{
								{
									Start: net.ParseIP("8.8.8.8"),
									End:   net.ParseIP("9.9.9.9"),
								},
							},
							Ports: []garden.PortRange{
								{
									Start: 53,
									End:   54,
								},
							},
						},
					},
				},
			},
		}

		input = GetInput(inputStruct)

		containerID = "some-container-id-that-is-long"
		netinChainName = ("netin--" + containerID)[:28]
		netoutChainName = ("netout--" + containerID)[:28]
		inputChainName = ("input--" + containerID)[:28]
		netoutLoggingChainName = fmt.Sprintf("%s--log", netoutChainName[:23])

		cmd = cniCommand("ADD", input)
	})

	AfterEach(func() {
		cmd := cniCommand("DEL", input)
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))

		By("checking that ip masquerade rule is removed")
		Expect(AllIPTablesRules("nat")).NotTo(ContainElement("-A POSTROUTING -s 1.2.3.4/32 ! -d 10.255.0.0/16 -j MASQUERADE"))

		By("checking that iptables netin rules are removed")
		Expect(AllIPTablesRules("nat")).ToNot(ContainElement(`-N ` + netinChainName))
		Expect(AllIPTablesRules("nat")).ToNot(ContainElement(`-A PREROUTING -j ` + netinChainName))

		By("checking that port forwarding rules were removed from the netin chain")
		Expect(AllIPTablesRules("nat")).ToNot(ContainElement("-A " + netinChainName + " -d 10.244.2.3/32 -p tcp -m tcp --dport 1000 -j DNAT --to-destination 1.2.3.4:1001"))
		Expect(AllIPTablesRules("nat")).ToNot(ContainElement("-A " + netinChainName + " -d 10.244.2.3/32 -p tcp -m tcp --dport 2000 -j DNAT --to-destination 1.2.3.4:2001"))

		By("checking that there are no more netout rules for this container")
		Expect(AllIPTablesRules("filter")).NotTo(ContainElement(ContainSubstring(inputChainName)))
		Expect(AllIPTablesRules("filter")).NotTo(ContainElement(ContainSubstring(netoutChainName)))
		Expect(AllIPTablesRules("filter")).NotTo(ContainElement(ContainSubstring(netoutLoggingChainName)))

		os.Remove(debugFileName)
		os.Remove(datastorePath)
		os.Remove(iptablesLockFilePath)

		healthCheckServer.Close()
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
			Expect(AllIPTablesRules("nat")).To(ContainElement("-A POSTROUTING -s 1.2.3.4/32 ! -d 10.255.0.0/16 -j MASQUERADE"))

			By("calling DEL")
			cmd = cniCommand("DEL", input)
			session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			By("check that ip masquerade rule is removed")
			Expect(AllIPTablesRules("nat")).NotTo(ContainElement("-A POSTROUTING -s 1.2.3.4/32 ! -d 10.255.0.0/16 -j MASQUERADE"))
		})
	})

	Context("When call with command ADD", func() {
		It("passes the delegate result back to the caller", func() {
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(MatchJSON(`{ "ips": [{ "version": "4", "interface": -1, "address": "1.2.3.4/32" }], "dns":{} }`))
		})

		It("passes the correct stdin to the delegate plugin", func() {
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			debug, err := noop_debug.ReadDebug(debugFileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(debug.Command).To(Equal("ADD"))

			Expect(debug.CmdArgs.StdinData).To(MatchJSON(`{
				"type": "noop",
				"some": "other data"
			}`))
		})

		It("ensures the container masquerade rule is created", func() {
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(MatchJSON(`{ "ips": [{ "version": "4", "interface": -1, "address": "1.2.3.4/32" }], "dns":{} }`))
			Expect(AllIPTablesRules("nat")).To(ContainElement("-A POSTROUTING -s 1.2.3.4/32 ! -d 10.255.0.0/16 -j MASQUERADE"))
		})

		Context("when no runtime config is passed in", func() {
			BeforeEach(func() {
				inputStruct.RuntimeConfig = nil
				input = GetInput(inputStruct)

				cmd = cniCommand("ADD", input)
			})
			It("does not write the default netout rules", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("checking that there are no netin rules")
				Expect(AllIPTablesRules("nat")).ToNot(ContainElement(`-N ` + netinChainName))
				Expect(AllIPTablesRules("nat")).ToNot(ContainElement(`-A PREROUTING -j ` + netinChainName))
				Expect(AllIPTablesRules("nat")).ToNot(ContainElement("-A " + netinChainName + " -d 10.244.2.3/32 -p tcp -m tcp --dport 1000 -j DNAT --to-destination 1.2.3.4:1001"))
				Expect(AllIPTablesRules("nat")).ToNot(ContainElement("-A " + netinChainName + " -d 10.244.2.3/32 -p tcp -m tcp --dport 2000 -j DNAT --to-destination 1.2.3.4:2001"))

				By("checking that there are no netout rules")
				Expect(AllIPTablesRules("filter")).NotTo(ContainElement(ContainSubstring(inputChainName)))
				Expect(AllIPTablesRules("filter")).NotTo(ContainElement(ContainSubstring(netoutChainName)))
				Expect(AllIPTablesRules("filter")).NotTo(ContainElement(ContainSubstring(netoutLoggingChainName)))
			})
		})

		Describe("PortMapping", func() {
			It("creates iptables portmapping rules", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("checking that a netin chain was created for the container")
				Expect(AllIPTablesRules("nat")).To(ContainElement(`-N ` + netinChainName))
				Expect(AllIPTablesRules("nat")).To(ContainElement(`-A PREROUTING -j ` + netinChainName))

				By("checking that port forwarding rules were added to the netin chain")
				Expect(AllIPTablesRules("nat")).To(ContainElement("-A " + netinChainName + " -d 10.244.2.3/32 -p tcp -m tcp --dport 1000 -j DNAT --to-destination 1.2.3.4:1001"))
				Expect(AllIPTablesRules("nat")).To(ContainElement("-A " + netinChainName + " -d 10.244.2.3/32 -p tcp -m tcp --dport 2000 -j DNAT --to-destination 1.2.3.4:2001"))
			})

			Context("when a port mapping with hostport 0 is given", func() {
				BeforeEach(func() {
					inputStruct.WrapperConfig.RuntimeConfig.PortMappings = []garden.NetIn{
						{
							HostPort:      0,
							ContainerPort: 1001,
						},
					}

					input = GetInput(inputStruct)
				})
				It("refuses to allocate", func() {
					cmd = cniCommand("ADD", input)
					session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(session).Should(gexec.Exit(1))
				})
			})

			Context("when adding netin rule fails", func() {
				BeforeEach(func() {
					inputStruct.WrapperConfig.InstanceAddress = "asdf"
					input = GetInput(inputStruct)
				})
				It("exit status 1", func() {
					cmd = cniCommand("ADD", input)
					session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(session).Should(gexec.Exit(1))
					Expect(session.Out.Contents()).To(MatchJSON(`{ "code": 100, "msg": "adding netin rule: invalid ip: asdf" }`))
				})
			})
		})

		Describe("NetOutRules", func() {
			It("creates iptables netout rules", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("checking that the default forwarding rules are created for that container")
				Expect(AllIPTablesRules("filter")).To(gomegamatchers.ContainSequence([]string{
					`-A ` + netoutChainName + ` -s 1.2.3.4/32 ! -d 10.255.0.0/16 -m state --state RELATED,ESTABLISHED -j RETURN`,
					`-A ` + netoutChainName + ` -s 1.2.3.4/32 ! -d 10.255.0.0/16 -j REJECT --reject-with icmp-port-unreachable`,
				}))

				By("checking that the default input rules are created for that container")
				Expect(AllIPTablesRules("filter")).To(gomegamatchers.ContainSequence([]string{
					`-A ` + inputChainName + ` -s 1.2.3.4/32 -m state --state RELATED,ESTABLISHED -j RETURN`,
					`-A ` + inputChainName + ` -s 1.2.3.4/32 -j REJECT --reject-with icmp-port-unreachable`,
				}))

				By("checking that the rules are written")
				Expect(AllIPTablesRules("filter")).To(ContainElement(`-A ` + netoutChainName + ` -s 1.2.3.4/32 -p tcp -m iprange --dst-range 8.8.8.8-9.9.9.9 -m tcp --dport 53:54 -j RETURN`))

			})

			Context("when iptables_asg_logging is enabled", func() {
				BeforeEach(func() {
					inputStruct.WrapperConfig.RuntimeConfig.NetOutRules[0].Log = false
					inputStruct.WrapperConfig.IPTablesASGLogging = true
					input = GetInput(inputStruct)
				})
				It("writes iptables asg logging rules", func() {
					cmd = cniCommand("ADD", input)
					session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(session).Should(gexec.Exit(0))

					By("checking that the filter rule was installed and that logging can be enabled")
					Expect(AllIPTablesRules("filter")).To(ContainElement(`-A ` + netoutChainName + ` -s 1.2.3.4/32 -p tcp -m iprange --dst-range 8.8.8.8-9.9.9.9 -m tcp --dport 53:54 -g ` + netoutLoggingChainName))

					By("checking that it writes the logging rules")
					Expect(AllIPTablesRules("filter")).To(ContainElement(`-A ` + netoutLoggingChainName + ` -p tcp -m conntrack --ctstate INVALID,NEW,UNTRACKED -j LOG --log-prefix OK_` + containerID[:26]))
				})
			})

			Context("when a rule has logging enabled", func() {
				BeforeEach(func() {
					inputStruct.WrapperConfig.RuntimeConfig.NetOutRules[0].Log = true
					inputStruct.WrapperConfig.IPTablesASGLogging = false
					input = GetInput(inputStruct)
				})
				It("writes iptables asg logging rules for that rule", func() {
					cmd = cniCommand("ADD", input)
					session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(session).Should(gexec.Exit(0))

					By("checking that the filter rule was installed and that logging can be enabled")
					Expect(AllIPTablesRules("filter")).To(ContainElement(`-A ` + netoutChainName + ` -s 1.2.3.4/32 -p tcp -m iprange --dst-range 8.8.8.8-9.9.9.9 -m tcp --dport 53:54 -g ` + netoutLoggingChainName))

					By("checking that it writes the logging rules")
					Expect(AllIPTablesRules("filter")).To(ContainElement(`-A ` + netoutLoggingChainName + ` -p tcp -m conntrack --ctstate INVALID,NEW,UNTRACKED -j LOG --log-prefix OK_` + containerID[:26]))
				})
			})
		})

		Context("When the health check call returns an error", func() {
			BeforeEach(func() {
				healthCheckServer.Close()
			})

			It("wraps and returns the error", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				var errData map[string]interface{}
				Expect(json.Unmarshal(session.Out.Contents(), &errData)).To(Succeed())
				Expect(errData["code"]).To(BeEquivalentTo(100))
				Expect(errData["msg"]).To(ContainSubstring("could not call health check: Get http"))
			})
		})

		Context("When the health check returns a non-200 status code", func() {
			BeforeEach(func() {
				healthCheckReturnStatus = 503
			})

			It("wraps and returns the error", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				var errData map[string]interface{}
				Expect(json.Unmarshal(session.Out.Contents(), &errData)).To(Succeed())
				Expect(errData["code"]).To(BeEquivalentTo(100))
				Expect(errData["msg"]).To(ContainSubstring("health check failed with 503"))
			})
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

		Context("when the container id is not specified", func() {
			BeforeEach(func() {
				cmd.Env[1] = "CNI_CONTAINERID="
			})

			It("wraps and returns the error", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				Expect(session.Out.Contents()).To(MatchJSON(`{ "code": 100, "msg": "initialize net out: invalid handle" }`))
			})

			It("does not leave any iptables rules behind", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				Expect(AllIPTablesRules("nat")).NotTo(ContainElement("-A POSTROUTING -s 1.2.3.4/32 ! -d 10.255.0.0/16 -j MASQUERADE"))
			})
		})

		Context("when the datastore add fails", func() {
			BeforeEach(func() {
				err := ioutil.WriteFile(datastorePath, []byte("banana"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())
			})

			It("wraps and returns the error", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				Expect(session.Out.Contents()).To(MatchJSON(`{ "code": 100, "msg": "store add: decoding file: invalid character 'b' looking for beginning of value" }`))
			})

			It("does not leave any iptables rules behind", func() {
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				Expect(AllIPTablesRules("nat")).NotTo(ContainElement("-A POSTROUTING -s 1.2.3.4/32 ! -d 10.255.0.0/16 -j MASQUERADE"))
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

			Expect(debug.CmdArgs.StdinData).To(MatchJSON(`{
				"type": "noop",
				"some": "other data"
			}`))
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

				Expect(debug.CmdArgs.StdinData).To(MatchJSON(`{
					"type": "noop",
					"some": "other data"
				}`))
			})
		})

	})
})
