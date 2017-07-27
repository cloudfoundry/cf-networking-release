package integration_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"policy-server/config"
	"policy-server/integration/helpers"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("External API Adding Policies", func() {
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
		dbConf.DatabaseName = fmt.Sprintf("test_node_%d", GinkgoParallelNode())
		testsupport.CreateDatabase(dbConf)

		template := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
		policyServerConfs = configurePolicyServers(template, 2)
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]
	})

	AfterEach(func() {
		stopPolicyServers(sessions)

		testsupport.RemoveDatabase(dbConf)

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Describe("adding policies", func() {
		addPoliciesSucceeds := func(headers map[string]string, request, expectedResponse string) {
			body := strings.NewReader(request)
			resp := helpers.MakeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
				headers,
				body,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(MatchJSON("{}"))

			resp = helpers.MakeAndDoRequest(
				"GET",
				fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
				headers,
				nil,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err = ioutil.ReadAll(resp.Body)
			Expect(responseString).To(MatchJSON(expectedResponse))

			Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
				HaveName("CreatePoliciesRequestTime"),
			))
			Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
				HaveName("StoreCreateSuccessTime"),
			))
		}

		addPoliciesFails := func(headers map[string]string, request, expectedResponse string) {
			body := strings.NewReader(request)
			resp := helpers.MakeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
				headers,
				body,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(MatchJSON(expectedResponse))

			Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
				HaveName("CreatePoliciesRequestTime"),
			))
		}

		v1Header := map[string]string{"Accept": "1.0.0"}
		v1Request := `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "ports": { "start": 8090, "end": 8090 } } } ] }`
		v1RequestMissingProtocol := `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "ports": { "start": 8080, "end": 8080 } } } ] }`
		v1Response := `{ "total_policies": 1, "policies": [ { "source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "ports": { "start": 8090, "end": 8090 } } } ]}`

		v0Header := map[string]string{"Accept": "0.0.0"}
		v0Request := `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8080 } } ] }`
		v0RequestMissingProtocol := `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "port": 8080 } } ] }`
		v0Response := `{ "total_policies": 1, "policies": [ { "source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8080 } } ]}`

		invalidStartPortResponse := `{ "error": "policies-create: invalid start port 0, must be in range 1-65535" }`
		invalidProtocolResponse := `{ "error": "policies-create: invalid destination protocol, specify either udp or tcp" }`

		DescribeTable("adding policies succeeds", addPoliciesSucceeds,
			Entry("v1", v1Header, v1Request, v1Response),
			Entry("v0", v0Header, v0Request, v0Response),
			Entry("no version", nil, v0Request, v0Response),
		)

		DescribeTable("failure cases", addPoliciesFails,
			Entry("v1: missing ports", v1Header, v0Request, invalidStartPortResponse),
			Entry("v1: missing protocol", v1Header, v1RequestMissingProtocol, invalidProtocolResponse),

			Entry("v0: missing port", v0Header, v1Request, invalidStartPortResponse),
			Entry("v0: missing protocol", v0Header, v0RequestMissingProtocol, invalidProtocolResponse),

			Entry("no version: missing port", nil, v1Request, invalidStartPortResponse),
			Entry("no version: missing protocol", nil, v0RequestMissingProtocol, invalidProtocolResponse),
		)
	})
})
