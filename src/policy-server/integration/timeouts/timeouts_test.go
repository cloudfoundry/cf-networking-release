package timeouts_test

import (
	"fmt"
	"math/rand"
	"net/http"
	"os/exec"
	"policy-server/config"
	"policy-server/integration/helpers"

	"code.cloudfoundry.org/go-db-helpers/metrics"
	"code.cloudfoundry.org/go-db-helpers/testsupport"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Timeout", func() {
	var (
		session          *gexec.Session
		conf             config.Config
		testDatabase     *testsupport.TestDatabase
		dbConnectionInfo *testsupport.DBConnectionInfo

		fakeMetron metrics.FakeMetron
	)
	BeforeEach(func() {

		fakeMetron = metrics.NewFakeMetron()

		dbName := fmt.Sprintf("test_netman_database_%x", rand.Int())
		dbConnectionInfo = testsupport.GetDBConnectionInfo()
		testDatabase = dbConnectionInfo.CreateDatabase(dbName)

		conf = helpers.DefaultTestConfig(testDatabase.DBConfig(), fakeMetron.Address(), "../fixtures")
		session = helpers.StartPolicyServer(policyServerPath, conf)
	})

	AfterEach(func() {
		session.Interrupt()
		Eventually(session, helpers.DEFAULT_TIMEOUT).Should(gexec.Exit())

		if testDatabase != nil {
			testDatabase.Destroy()
		}

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Context("when the database is unreachable", func() {
		BeforeEach(func() {
			mustSucceed("iptables", "-A", "INPUT", "-p", "tcp", "--dport", dbConnectionInfo.Port, "-j", "DROP")
		})
		AfterEach(func() {
			mustSucceed("iptables", "-D", "INPUT", "-p", "tcp", "--dport", dbConnectionInfo.Port, "-j", "DROP")
		})

		PIt("times out requests", func(done Done) {
			Expect(true).To(BeTrue())
			By("getting the policies")
			policyServerURL := fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort)
			fmt.Println("starting")
			resp := helpers.MakeAndDoRequest("GET", policyServerURL, nil)
			fmt.Println("done")
			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
		}, 5 /* timeout for It block, in seconds */)

	})

})

func mustSucceed(binary string, args ...string) string {
	cmd := exec.Command(binary, args...)
	sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, helpers.DEFAULT_TIMEOUT).Should(gexec.Exit(0))
	return string(sess.Out.Contents())
}
