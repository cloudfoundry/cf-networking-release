package integration_test

import (
	"fmt"
	"io/ioutil"
	"lib/testsupport"
	"math/rand"
	"net/http"
	"netmon/integration/fakes"
	"policy-server/config"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Automatic Stale Policy Cleanup", func() {
	var (
		sessions          []*gexec.Session
		conf              config.Config
		policyServerConfs []config.Config
		testDatabase      *testsupport.TestDatabase

		fakeMetron fakes.FakeMetron
	)

	BeforeEach(func() {
		fakeMetron = fakes.New()

		dbName := fmt.Sprintf("test_netman_database_%x", rand.Int())
		dbConnectionInfo := testsupport.GetDBConnectionInfo()
		testDatabase = dbConnectionInfo.CreateDatabase(dbName)

		template := DefaultTestConfig(testDatabase.DBConfig(), fakeMetron.Address())
		template.CleanupInterval = 1
		template.CCAppRequestChunkSize = 1

		policyServerConfs = configurePolicyServers(template, 2)
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]
	})

	AfterEach(func() {
		for _, session := range sessions {
			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		}

		if testDatabase != nil {
			testDatabase.Destroy()
		}

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Describe("Automatic Stale Policy Cleanup", func() {
		BeforeEach(func() {
			body := strings.NewReader(`{ "policies": [
				{"source": { "id": "live-app-1-guid" }, "destination": { "id": "live-app-2-guid", "protocol": "tcp", "port": 8080 } },
				{"source": { "id": "live-app-2-guid" }, "destination": { "id": "live-app-2-guid", "protocol": "tcp", "port": 9999 } },
				{"source": { "id": "live-app-1-guid" }, "destination": { "id": "dead-app", "protocol": "tcp", "port": 3333 } }
				]} `)

			resp := makeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
				body,
			)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("eventually cleans up stale policies stale policies", func() {
			listPolicies := func() []byte {
				resp := makeAndDoRequest(
					"GET",
					fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
					nil,
				)
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				bodyBytes, _ := ioutil.ReadAll(resp.Body)
				return bodyBytes
			}

			activePolicies := `{ "total_policies": 2,
			"policies": [
				{"source": { "id": "live-app-1-guid" }, "destination": { "id": "live-app-2-guid", "protocol": "tcp", "port": 8080 } },
				{"source": { "id": "live-app-2-guid" }, "destination": { "id": "live-app-2-guid", "protocol": "tcp", "port": 9999 } }
				]} `
			Eventually(listPolicies, "5s").Should(MatchJSON(activePolicies))

			By("emitting store metrics")
			Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
				HaveName("StoreDeleteTime"),
			))
		})
	})
})
