package integration_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"policy-server/config"
	"policy-server/integration/helpers"
	"strings"

	"code.cloudfoundry.org/go-db-helpers/db"
	"code.cloudfoundry.org/go-db-helpers/metrics"
	"code.cloudfoundry.org/go-db-helpers/testsupport"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Automatic Stale Policy Cleanup", func() {
	var (
		sessions          []*gexec.Session
		conf              config.Config
		policyServerConfs []config.Config
		dbConf            db.Config

		fakeMetron metrics.FakeMetron
	)

	BeforeEach(func() {
		fakeMetron = metrics.NewFakeMetron()

		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("test_%x", rand.Int())
		testsupport.CreateDatabase(dbConf)

		template := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
		template.CleanupInterval = 1
		template.CCAppRequestChunkSize = 1

		policyServerConfs = configurePolicyServers(template, 2)
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]
	})

	AfterEach(func() {
		for _, session := range sessions {
			session.Interrupt()
			Eventually(session, helpers.DEFAULT_TIMEOUT).Should(gexec.Exit())
		}

		testsupport.RemoveDatabase(dbConf)

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Describe("Automatic Stale Policy Cleanup", func() {
		BeforeEach(func() {
			body := strings.NewReader(`{ "policies": [
				{"source": { "id": "live-app-1-guid" }, "destination": { "id": "live-app-2-guid", "protocol": "tcp", "port": 8080 } },
				{"source": { "id": "live-app-2-guid" }, "destination": { "id": "live-app-2-guid", "protocol": "tcp", "port": 9999 } },
				{"source": { "id": "live-app-1-guid" }, "destination": { "id": "dead-app", "protocol": "tcp", "port": 3333 } }
				]} `)

			resp := helpers.MakeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
				body,
			)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("eventually cleans up stale policies stale policies", func() {
			listPolicies := func() []byte {
				resp := helpers.MakeAndDoRequest(
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
