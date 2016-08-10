package acceptance_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
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
		originalRunningSecurityGroups []string
	)

	BeforeEach(func() {
		appA = fmt.Sprintf("appA-%d", rand.Int31())

		Auth(testConfig.TestUser, testConfig.TestUserPassword)

		orgName = "test-org"
		Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

		spaceName = "test-space"
		Expect(cf.Cf("create-space", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))

		pushApp(appA)
		appRoute = fmt.Sprintf("http://%s.%s/", appA, config.AppsDomain)

		allSecurityGroups := getAllSecurityGroups()
		for _, sg := range allSecurityGroups {
			Expect(cf.Cf("bind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
		}
		originalRunningSecurityGroups = getRunningSecurityGroups()
	})

	AfterEach(func() {
		AppReport(appA, Timeout_Short)

		// clean up everything
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))

		By("adding back all the original security groups")
		for _, sg := range originalRunningSecurityGroups {
			Expect(cf.Cf("bind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
		}
	})

	Describe("accessing the app via the router", func() {
		It("allows external traffic to reach the app", func(done Done) {
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

			close(done)
		}, 60 /* <-- overall spec timeout in seconds */)
	})

	Describe("reaching the internet", func() {
		It("can reach the internet only if the application security groups allow it", func() {
			By("checking that the internet is reachable")
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

			By("removing all the original security groups")
			for _, sg := range originalRunningSecurityGroups {
				Expect(cf.Cf("unbind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
			}

			By("restarting the app")
			Expect(cf.Cf("restart", appA).Wait(Timeout_Push)).To(gexec.Exit(0))

			By("checking that the connection fails")
			Consistently(func() bool {
				resp, err := http.Get(appRoute + "proxy/example.com")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(500))
				respBytes, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(respBytes).To(ContainSubstring("example.com"))
				return true
			}, "10s", "1s").Should(BeTrue())
		})
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
