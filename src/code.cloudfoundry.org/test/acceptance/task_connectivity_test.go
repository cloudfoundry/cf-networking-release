package acceptance_test

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

const Timeout_Task_Curl = 1 * time.Minute

type ProxyResponse struct {
	ListenAddresses []string ""
	Port            int
}

var _ = Describe("task connectivity on the overlay network", func() {
	Describe("networking policy", func() {
		var (
			prefix  string
			domain  string
			orgName string
			proxy1  string
			proxy2  string
		)

		BeforeEach(func() {
			prefix = testConfig.Prefix
			domain = config.AppsDomain

			orgName = prefix + "task-org"
			Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
			Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

			spaceName := prefix + "space"
			Expect(cf.Cf("create-space", spaceName, "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
			Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))

			proxy1 = "proxy-task-connectivity-1"
			proxy2 = "proxy-task-connectivity-2"

			pushProxy(proxy1)
			pushProxy(proxy2)

			cfCLI.AddNetworkPolicy(proxy1, proxy2, 8080, "tcp")
		})

		AfterEach(func() {
			Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
			_, err := cfCLI.CleanupStaleNetworkPolicies()
			Expect(err).NotTo(HaveOccurred())
		})

		It("allows tasks to talk to app instances", func(ctx SpecContext) {
			By("getting the overlay ip of proxy2")
			cmd := exec.Command("curl", "--fail", proxy2+"."+domain)
			sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, 5*time.Second).Should(gexec.Exit(0))

			var proxy2Response ProxyResponse
			Expect(json.Unmarshal(sess.Out.Contents(), &proxy2Response)).To(Succeed())

			containerIP := getContainerIP(proxy2Response.ListenAddresses)

			By("Checking that the task associated with proxy1 can connect to proxy2")
			commandToRun := `
			while true; do
				if curl --fail "` + containerIP + `:` + strconv.Itoa(proxy2Response.Port) + `" ; then
					exit 0
				fi
			done;
			exit 1
			`
			Expect(cfCLI.RunTask(proxy1, commandToRun)).To(Succeed())

			Eventually(func() *gbytes.Buffer {
				return cf.Cf("tasks", proxy1).Wait(10 * time.Second).Out
			}, Timeout_Task_Curl).Should(gbytes.Say("SUCCEEDED"))
		}, SpecTimeout(30*time.Minute))
	})
})

func getContainerIP(listenAddresses []string) string {
	for _, listenAddr := range listenAddresses {
		if !strings.HasPrefix(listenAddr, "127.0.0.1") {
			return listenAddr
		}
	}

	return ""
}
