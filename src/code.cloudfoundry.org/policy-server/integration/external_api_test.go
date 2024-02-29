package integration_test

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	"code.cloudfoundry.org/policy-server/config"
	"code.cloudfoundry.org/policy-server/integration/helpers"
	testhelpers "code.cloudfoundry.org/test-helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("External API", func() {
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
		dbConf.DatabaseName = fmt.Sprintf("external_api_test_node_%d", ports.PickAPort())

		template, _, _ := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
		policyServerConfs = configurePolicyServers(template, 2)
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]
	})

	AfterEach(func() {
		stopPolicyServers(sessions, policyServerConfs)

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Describe("authentication", func() {
		var makeNewRequest = func(method, route, bodyString string) *http.Request {
			var body io.Reader
			if bodyString != "" {
				body = strings.NewReader(bodyString)
			}
			url := fmt.Sprintf("http://%s:%d/%s", conf.ListenHost, conf.ListenPort, route)
			req, err := http.NewRequest(method, url, body)
			Expect(err).NotTo(HaveOccurred())

			return req
		}

		var TestMissingAuthHeader = func(req *http.Request) {
			By("check that 401 is returned when auth header is missing")
			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			responseString, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(MatchJSON(`{ "error": "missing authorization header"}`))
		}

		var TestBadBearerToken = func(req *http.Request) {
			By("check that 401 is returned when auth header is invalid")
			req.Header.Set("Authorization", "Bearer bad-token")

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			responseString, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(MatchJSON(`{ "error": "failed to verify token with uaa" }`))
		}

		var _ = DescribeTable("all the routes",
			func(method, route, bodyString string) {
				TestMissingAuthHeader(makeNewRequest(method, route, bodyString))
				TestBadBearerToken(makeNewRequest(method, route, bodyString))
			},
			Entry("POST to policies",
				"POST",
				"networking/v1/external/policies",
				`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "ports": { "start": 8090, "end": 8090 } } } ] }`,
			),
			Entry("GET to policies",
				"GET",
				"networking/v1/external/policies",
				``,
			),
			Entry("POST to policies/delete",
				"POST",
				"networking/v1/external/policies/delete",
				`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "ports": { "start": 8090, "end": 8090 } } } ] }`,
			),
		)
	})

	Describe("uptime", func() {
		It("returns 200 when server is healthy", func() {
			resp := helpers.MakeAndDoRequest(
				"GET",
				fmt.Sprintf("http://%s:%d/", conf.ListenHost, conf.ListenPort),
				nil,
				nil,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		Context("when the database is unavailable", func() {
			BeforeEach(func() {
				testhelpers.RemoveDatabase(dbConf)
			})

			It("still returns a 200", func() {
				resp := helpers.MakeAndDoRequest(
					"GET",
					fmt.Sprintf("http://%s:%d/", conf.ListenHost, conf.ListenPort),
					nil,
					nil,
				)

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("health", func() {
		It("returns 200 when server is healthy", func() {
			resp := helpers.MakeAndDoRequest(
				"GET",
				fmt.Sprintf("http://%s:%d/health", conf.ListenHost, conf.ListenPort),
				nil,
				nil,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})
})
