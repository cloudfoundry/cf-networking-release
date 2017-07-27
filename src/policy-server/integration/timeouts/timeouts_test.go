package timeouts_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"policy-server/config"
	"policy-server/integration/helpers"
	"strconv"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const testTimeoutInSeconds = 5

var policiesBody = `{
	"policies": [{
		"source": { "id": "some-app-guid" },
		"destination": { "id": "some-other-app-guid", "protocol": "tcp", "ports": {"start": 8090, "end": 8090} }
	}]
}`

var _ = Describe("Timeout", func() {
	var (
		session *gexec.Session
		conf    config.Config
		dbConf  db.Config
		headers map[string]string

		fakeMetron      metrics.FakeMetron
		policyServerURL string
	)
	BeforeEach(func() {
		dbConf = testsupport.GetDBConfig()
		if dbConf.Type == "postgres" {
			Skip("skipping timeout tests on postgres; only supported by mysql")
		}

		dbConf.DatabaseName = fmt.Sprintf("test_timeouts_node_%d", GinkgoParallelNode())
		dbConf.Timeout = 1
		testsupport.CreateDatabase(dbConf)

		fakeMetron = metrics.NewFakeMetron()

		conf = helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "../fixtures")
		session = helpers.StartPolicyServer(policyServerPath, conf)
		policyServerURL = fmt.Sprintf("http://%s:%d", conf.ListenHost, conf.ListenPort)

		resp := helpers.MakeAndDoRequest("GET", fmt.Sprintf("%s/%s", policyServerURL, "networking/v0/external/policies"), headers, nil)
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(ioutil.ReadAll(resp.Body)).To(MatchJSON(`{ "total_policies": 0, "policies": [] }`))
	})

	AfterEach(func() {
		session.Interrupt()
		Eventually(session, helpers.DEFAULT_TIMEOUT).Should(gexec.Exit())

		testsupport.RemoveDatabase(dbConf)

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Context("when the database is unreachable", func() {
		BeforeEach(func() {
			By("blocking access to port " + strconv.Itoa(int(dbConf.Port)))
			mustSucceed("iptables", "-w", "-A", "INPUT", "-p", "tcp", "--dport", strconv.Itoa(int(dbConf.Port)), "-j", "DROP")
		})
		AfterEach(func() {
			By("allowing access to port " + strconv.Itoa(int(dbConf.Port)))
			mustSucceed("iptables", "-w", "-D", "INPUT", "-p", "tcp", "--dport", strconv.Itoa(int(dbConf.Port)), "-j", "DROP")
		})

		itTimesOut := func(description string, endpointMethod string, endpointPath string, bodyString string, failureJSON string) {
			It(fmt.Sprintf("times out %s", description), func(done Done) {
				var body io.Reader
				if bodyString != "" {
					body = strings.NewReader(bodyString)
				}
				resp := helpers.MakeAndDoRequest(
					endpointMethod,
					fmt.Sprintf("%s/%s", policyServerURL, endpointPath),
					headers,
					body,
				)
				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
				Expect(ioutil.ReadAll(resp.Body)).To(MatchJSON(failureJSON))

				close(done)
			}, float64(testTimeoutInSeconds))
		}

		itTimesOut("getting policies",
			"GET", "networking/v0/external/policies", "",
			`{ "error": "policies-index: database read failed" }`,
		)
		itTimesOut("creating policies",
			"POST", "networking/v0/external/policies", policiesBody,
			`{ "error": "policies-create: database create failed" }`,
		)
		itTimesOut("deleting policies",
			"POST", "networking/v0/external/policies/delete", policiesBody,
			`{ "error": "delete-policies: database delete failed" }`,
		)
		itTimesOut("getting tags",
			"GET", "networking/v0/external/tags", "",
			`{ "error": "tags-index: database read failed" }`,
		)
		itTimesOut("cleaning up",
			"POST", "networking/v0/external/policies/cleanup", "",
			`{ "error": "policies-cleanup: policies cleanup failed" }`,
		)
		itTimesOut("checking health",
			"GET", "health", "",
			`{ "error": "health: check database failed" }`,
		)
	})
})

func mustSucceed(binary string, args ...string) string {
	cmd := exec.Command(binary, args...)
	sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, helpers.DEFAULT_TIMEOUT).Should(gexec.Exit(0))
	return string(sess.Out.Contents())
}
