package timeouts_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os/exec"
	"policy-server/config"
	"policy-server/integration/helpers"
	"time"

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

		fakeMetron      metrics.FakeMetron
		policyServerURL string
	)
	BeforeEach(func() {
		fakeMetron = metrics.NewFakeMetron()

		dbName := fmt.Sprintf("test_netman_database_%x", rand.Int())
		dbConnectionInfo = testsupport.GetDBConnectionInfo()
		dbConnectionInfo.ConnectTimeout = 1 * time.Second
		dbConnectionInfo.ReadTimeout = 1 * time.Second
		testDatabase = dbConnectionInfo.CreateDatabase(dbName)

		conf = helpers.DefaultTestConfig(testDatabase.DBConfig(), fakeMetron.Address(), "../fixtures")
		session = helpers.StartPolicyServer(policyServerPath, conf)
		policyServerURL = fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort)

		resp := helpers.MakeAndDoRequest("GET", policyServerURL, nil)
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(ioutil.ReadAll(resp.Body)).To(MatchJSON(`{ "total_policies": 0, "policies": [] }`))

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
			By("blocking access to port " + dbConnectionInfo.Port)
			mustSucceed("iptables", "-A", "INPUT", "-p", "tcp", "--dport", dbConnectionInfo.Port, "-j", "DROP")
		})
		AfterEach(func() {
			By("allowing access to port " + dbConnectionInfo.Port)
			mustSucceed("iptables", "-D", "INPUT", "-p", "tcp", "--dport", dbConnectionInfo.Port, "-j", "DROP")
		})

		It("times out requests", func(done Done) {
			resp := helpers.MakeAndDoRequest("GET", policyServerURL, nil)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
			Expect(ioutil.ReadAll(resp.Body)).To(MatchJSON(`{ "error": "policies-index: database read failed" }`))

			close(done)
		}, 3 /* timeout for It block, in seconds */)
	})

})

func mustSucceed(binary string, args ...string) string {
	cmd := exec.Command(binary, args...)
	sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, helpers.DEFAULT_TIMEOUT).Should(gexec.Exit(0))
	return string(sess.Out.Contents())
}
