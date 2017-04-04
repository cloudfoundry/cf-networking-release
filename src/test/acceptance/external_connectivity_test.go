package acceptance_test

import (
	"cf-pusher/cf_cli_adapter"
	"fmt"
	"io/ioutil"
	"lib/testsupport"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("external connectivity", func() {
	var (
		appA                          string
		orgName                       string
		spaceName                     string
		appRoute                      string
		prefix                        string
		originalRunningSecurityGroups []string
		cli                           *cf_cli_adapter.Adapter
		tcpASGFile                    string
		udpASGFile                    string
		icmpASGFile                   string
	)

	BeforeEach(func() {
		cli = &cf_cli_adapter.Adapter{
			CfCliPath: "cf",
		}
		appA = fmt.Sprintf("appA-%d", rand.Int31())
		prefix = testConfig.Prefix

		AuthAsAdmin()

		orgName = prefix + "external-connectivity-org"
		Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

		spaceName = prefix + "space"
		Expect(cf.Cf("create-space", spaceName, "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))

		pushProxy(appA)
		appRoute = fmt.Sprintf("http://%s.%s/", appA, config.AppsDomain)

		allSecurityGroups := getAllSecurityGroups()
		for _, sg := range allSecurityGroups {
			Expect(cf.Cf("bind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
		}

		originalRunningSecurityGroups = getRunningSecurityGroups()
	})

	AfterEach(func() {
		appReport(appA, Timeout_Short)
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))

		By("adding back all the original security groups", func() {
			for _, sg := range originalRunningSecurityGroups {
				Expect(cf.Cf("bind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
			}
		})
		os.Remove(tcpASGFile)
		os.Remove(udpASGFile)
		os.Remove(icmpASGFile)
	})

	Describe("basic (legacy) network behavior for an app", func() {
		It("is reachable from the router, and can reach the internet only if allowed", func(done Done) {
			By("checking that the app is reachable via the router", func() {
				Consistently(func() bool {
					resp, err := http.Get(appRoute)
					Expect(err).NotTo(HaveOccurred())
					defer resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(200))
					respBytes, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(respBytes).To(ContainSubstring(`{"ListenAddresses":[`))
					return true
				}, "10s", "1s").Should(BeTrue())
			})

			By("checking that the app can reach the internet", func() {
				Consistently(func() bool {
					resp, err := http.Get(appRoute + "proxy/example.com")
					Expect(err).NotTo(HaveOccurred())
					defer resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(200))
					respBytes, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(respBytes).To(ContainSubstring("Example Domain"))
					return true
				}, "10s", "1s").Should(BeTrue())
			})

			By("checking that the app can ping the internet", func() {
				Consistently(func() bool {
					resp, err := http.Get(appRoute + "ping/example.com")
					Expect(err).NotTo(HaveOccurred())
					defer resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(200))
					respBytes, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(respBytes).To(ContainSubstring("Ping succeeded"))
					return true
				}, "10s", "1s").Should(BeTrue())
			})

			By("removing all the original security groups", func() {
				for _, sg := range originalRunningSecurityGroups {
					Expect(cf.Cf("unbind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
				}
			})

			By("restarting the app", func() {
				Expect(cf.Cf("restart", appA).Wait(Timeout_Push)).To(gexec.Exit(0))
			})

			By("checking that the app cannot reach the internet using http and dns", func() {
				Consistently(func() bool {
					resp, err := http.Get(appRoute + "proxy/example.com")
					Expect(err).NotTo(HaveOccurred())
					defer resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(500))
					respBytes, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(respBytes).To(ContainSubstring("example.com"))
					return true
				}, "5s", "1s").Should(BeTrue())
			})

			By("checking that the app cannot ping the internet", func() {
				Consistently(func() bool {
					resp, err := http.Get(appRoute + "ping/example.com")
					Expect(err).NotTo(HaveOccurred())
					defer resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(500))
					respBytes, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(respBytes).To(ContainSubstring("Ping failed to destination: example.com"))
					return true
				}, "5s", "1s").Should(BeTrue())
			})

			By("creating and binding a tcp and udp security group", func() {
				var err error
				tcpASGFile, err = testsupport.CreateASGFile(tcpASG())
				Expect(err).NotTo(HaveOccurred())
				Expect(cli.CreateSecurityGroup("tcp-asg", tcpASGFile)).To(Succeed())
				Expect(cli.BindSecurityGroup("tcp-asg", orgName, spaceName)).To(Succeed())

				udpASGFile, err = testsupport.CreateASGFile(udpASG())
				Expect(err).NotTo(HaveOccurred())
				Expect(cli.CreateSecurityGroup("udp-asg", udpASGFile)).To(Succeed())
				Expect(cli.BindSecurityGroup("udp-asg", orgName, spaceName)).To(Succeed())
			})

			By("restarting the app", func() {
				Expect(cf.Cf("restart", appA).Wait(Timeout_Push)).To(gexec.Exit(0))
			})

			By("checking that the app can use dns and http to reach the internet", func() {
				Consistently(func() bool {
					resp, err := http.Get(appRoute + "proxy/example.com")
					Expect(err).NotTo(HaveOccurred())
					defer resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(200))
					respBytes, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(respBytes).To(ContainSubstring("Example Domain"))
					return true
				}, "5s", "1s").Should(BeTrue())
			})

			By("checking that the app cannot ping the internet", func() {
				Consistently(func() bool {
					resp, err := http.Get(appRoute + "ping/example.com")
					Expect(err).NotTo(HaveOccurred())
					defer resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(500))
					respBytes, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(respBytes).To(ContainSubstring("Ping failed to destination: example.com"))
					return true
				}, "5s", "1s").Should(BeTrue())
			})

			By("creating and binding an icmp security group", func() {
				var err error
				icmpASGFile, err = testsupport.CreateASGFile(icmpASG())
				Expect(err).NotTo(HaveOccurred())
				Expect(cli.CreateSecurityGroup("icmp-asg", icmpASGFile)).To(Succeed())
				Expect(cli.BindSecurityGroup("icmp-asg", orgName, spaceName)).To(Succeed())
			})

			By("removing the tcp security groups", func() {
				Expect(cli.UnbindSecurityGroup("tcp-asg", orgName, spaceName)).To(Succeed())
			})

			By("restarting the app", func() {
				Expect(cf.Cf("restart", appA).Wait(Timeout_Push)).To(gexec.Exit(0))
			})

			By("checking that the app cannot use http to reach the internet", func() {
				Consistently(func() bool {
					resp, err := http.Get(appRoute + "proxy/example.com")
					Expect(err).NotTo(HaveOccurred())
					defer resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(500))
					respBytes, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(respBytes).To(ContainSubstring("example.com"))
					return true
				}, "5s", "1s").Should(BeTrue())
			})

			By("checking that the app can ping the internet", func() {
				Consistently(func() bool {
					resp, err := http.Get(appRoute + "ping/example.com")
					Expect(err).NotTo(HaveOccurred())
					defer resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(200))
					respBytes, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(respBytes).To(ContainSubstring("Ping succeeded"))
					return true
				}, "5s", "1s").Should(BeTrue())
			})

			close(done)
		}, 180 /* <-- overall spec timeout in seconds */)
	})
})

func getRunningSecurityGroups() []string {
	session := cf.Cf("running-security-groups")
	Expect(session.Wait(Timeout_Short)).To(gexec.Exit(0))

	candidateGroups := strings.Split(string(session.Out.Contents()), "\n")[3:]
	actualGroups := []string{}
	for _, l := range candidateGroups {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			actualGroups = append(actualGroups, trimmed)
		}
	}
	return actualGroups
}

func getAllSecurityGroups() []string {
	session := cf.Cf("security-groups")
	Expect(session.Wait(Timeout_Short)).To(gexec.Exit(0))

	candidateGroups := strings.Split(string(session.Out.Contents()), "\n")[4:]
	actualGroups := []string{}
	for _, l := range candidateGroups {
		fields := strings.Fields(l)
		if len(fields) < 2 {
			continue
		}
		trimmed := strings.TrimSpace(fields[1])
		if trimmed != "" {
			actualGroups = append(actualGroups, trimmed)
		}
	}
	return actualGroups
}

func tcpASG() string {
	return `
	[
		{
			"destination": "0.0.0.0-9.255.255.255",
			"protocol": "tcp",
			"ports": "80"
		},
		{
			"destination": "11.0.0.0-169.253.255.255",
			"protocol": "tcp",
			"ports": "80"
		},
		{
			"destination": "169.255.0.0-172.15.255.255",
			"protocol": "tcp",
			"ports": "80"
		},
		{
			"destination": "172.32.0.0-192.167.255.255",
			"protocol": "tcp",
			"ports": "80"
		},
		{
			"destination": "192.169.0.0-255.255.255.255",
			"protocol": "tcp",
			"ports": "80"
		}
	]
	`
}

func udpASG() string {
	return `
	[
		{
			"destination": "0.0.0.0-9.255.255.255",
			"protocol": "udp",
			"ports": "53"
		},
		{
			"destination": "11.0.0.0-169.253.255.255",
			"protocol": "udp",
			"ports": "53"
		},
		{
			"destination": "169.255.0.0-172.15.255.255",
			"protocol": "udp",
			"ports": "53"
		},
		{
			"destination": "172.32.0.0-192.167.255.255",
			"protocol": "udp",
			"ports": "53"
		},
		{
			"destination": "192.169.0.0-255.255.255.255",
			"protocol": "udp",
			"ports": "53"
		}
	]
	`
}

func icmpASG() string {
	return `
	[
		{
			"destination": "0.0.0.0-9.255.255.255",
			"protocol": "icmp",
			"type": 8,
			"code": 0
		},
		{
			"destination": "11.0.0.0-169.253.255.255",
			"protocol": "icmp",
			"type": 8,
			"code": 0
		},
		{
			"destination": "169.255.0.0-172.15.255.255",
			"protocol": "icmp",
			"type": 8,
			"code": 0
		},
		{
			"destination": "172.32.0.0-192.167.255.255",
			"protocol": "icmp",
			"type": 8,
			"code": 0
		},
		{
			"destination": "192.169.0.0-255.255.255.255",
			"protocol": "icmp",
			"type": 8,
			"code": 0
		}
	]
	`
}
