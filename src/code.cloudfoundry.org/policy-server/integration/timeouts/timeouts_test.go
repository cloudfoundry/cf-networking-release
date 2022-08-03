package timeouts_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	"code.cloudfoundry.org/policy-server/config"
	"code.cloudfoundry.org/policy-server/integration/helpers"
	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/store/migrations"
	testhelpers "code.cloudfoundry.org/test-helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"code.cloudfoundry.org/lager"
)

const testTimeoutInSeconds = 5

var policiesBodyV0 = `{
	"policies": [{
		"source": { "id": "some-app-guid" },
		"destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 1234 }
	}]
}`

var policiesBodyV1 = `{
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
		dbConf.DatabaseName = fmt.Sprintf("test_timeouts_node_%d", ports.PickAPort())
		dbConf.Timeout = 1
		testhelpers.CreateDatabase(dbConf)

		migrateAndPopulateTags(dbConf)

		fakeMetron = metrics.NewFakeMetron()

		conf, _, _ = helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "../fixtures")
		session = helpers.StartPolicyServer(policyServerPath, conf)
		policyServerURL = fmt.Sprintf("http://%s:%d", conf.ListenHost, conf.ListenPort)

		resp := helpers.MakeAndDoRequest("GET", fmt.Sprintf("%s/%s", policyServerURL, "networking/v1/external/policies"), headers, nil)
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(ioutil.ReadAll(resp.Body)).To(MatchJSON(`{ "total_policies": 0, "policies": [] }`))
	})

	AfterEach(func() {
		session.Interrupt()
		Eventually(session, helpers.DEFAULT_TIMEOUT).Should(gexec.Exit())

		testhelpers.RemoveDatabase(dbConf)

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
			done := make(chan interface{})
			timeout := float64(testTimeoutInSeconds)
			defer GinkgoRecover()
			go func() {
				It(fmt.Sprintf("times out %s", description), func() {
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
				})
				close(done)
			}()
			Eventually(done, timeout).Should(BeClosed())
		}

		// v1
		itTimesOut("V1 getting policies",
			"GET", "networking/v1/external/policies", "",
			`{ "error": "database read failed" }`,
		)
		itTimesOut("V1 creating policies",
			"POST", "networking/v1/external/policies", policiesBodyV1,
			`{ "error": "database create failed" }`,
		)
		itTimesOut("V1 deleting policies",
			"POST", "networking/v1/external/policies/delete", policiesBodyV1,
			`{ "error": "database delete failed" }`,
		)
		itTimesOut("V1 getting tags",
			"GET", "networking/v1/external/tags", "",
			`{ "error": "database read failed" }`,
		)
		itTimesOut("V1 cleaning up",
			"POST", "networking/v1/external/policies/cleanup", "",
			`{ "error": "policies cleanup failed" }`,
		)

		// v0
		itTimesOut("V0 getting policies",
			"GET", "networking/v0/external/policies", "",
			`{ "error": "database read failed" }`,
		)
		itTimesOut("V0 creating policies",
			"POST", "networking/v0/external/policies", policiesBodyV0,
			`{ "error": "database create failed" }`,
		)
		itTimesOut("V0 deleting policies",
			"POST", "networking/v0/external/policies/delete", policiesBodyV0,
			`{ "error": "database delete failed" }`,
		)
		itTimesOut("V0 getting tags",
			"GET", "networking/v0/external/tags", "",
			`{ "error": "database read failed" }`,
		)
		itTimesOut("V0 cleaning up",
			"POST", "networking/v0/external/policies/cleanup", "",
			`{ "error": "policies cleanup failed" }`,
		)

		itTimesOut("checking health",
			"GET", "health", "",
			`{ "error": "check database failed" }`,
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

func migrateAndPopulateTags(dbConf db.Config) {
	logger := lager.NewLogger("Timeout Test")

	realDb, err := db.NewConnectionPool(dbConf, 200, 0, 60*time.Minute, "Timeout Test", "Timeout Test", logger)
	Expect(err).NotTo(HaveOccurred())
	defer realDb.Close()

	migrator := &migrations.Migrator{
		MigrateAdapter: &migrations.MigrateAdapter{},
		MigrationsProvider: &migrations.MigrationsProvider{
			Store: &store.MigrationsStore{
				DBConn: realDb,
			},
		},
	}
	_, err = migrator.PerformMigrations(realDb.DriverName(), realDb, 0)
	Expect(err).ToNot(HaveOccurred())

	tagPopulator := &store.TagPopulator{DBConnection: realDb}
	err = tagPopulator.PopulateTables(1)
	Expect(err).NotTo(HaveOccurred())
}
