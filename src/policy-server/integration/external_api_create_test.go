package integration_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"policy-server/config"
	"policy-server/integration/helpers"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/db"
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

		fakeMetron testsupport.FakeMetron
	)

	BeforeEach(func() {
		fakeMetron = testsupport.NewFakeMetron()

		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("external_api_create_test_node_%d", testsupport.PickAPort())

		template, _ := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
		policyServerConfs = configurePolicyServers(template, 2)
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]
	})

	AfterEach(func() {
		stopPolicyServers(sessions, policyServerConfs, nil)

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Describe("adding policies", func() {
		addPoliciesSucceeds := func(version, request, expectedResponse string) {
			body := strings.NewReader(request)
			resp := helpers.MakeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/%s/external/policies", conf.ListenHost, conf.ListenPort, version),
				nil,
				body,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(MatchJSON("{}"))

			resp = helpers.MakeAndDoRequest(
				"GET",
				fmt.Sprintf("http://%s:%d/networking/%s/external/policies", conf.ListenHost, conf.ListenPort, version),
				nil,
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

		addPoliciesFails := func(version, request, expectedResponse string) {
			body := strings.NewReader(request)
			resp := helpers.MakeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/%s/external/policies", conf.ListenHost, conf.ListenPort, version),
				nil,
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

		v1Request := `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "ports": { "start": 8090, "end": 8090 } } } ] }`
		v1RequestMissingProtocol := `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "ports": { "start": 8080, "end": 8080 } } } ] }`
		v1Response := `{ "total_policies": 1, "policies": [ { "source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "ports": { "start": 8090, "end": 8090 } } } ]}`

		v0Request := `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8080 } } ] }`
		v0RequestMissingProtocol := `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "port": 8080 } } ] }`
		v0Response := `{ "total_policies": 1, "policies": [ { "source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8080 } } ]}`

		missingStartPortResponse := `{ "error": "mapper: validate policies: missing start port" }`
		missingPortResponse := `{ "error": "mapper: validate policies: missing port" }`
		invalidProtocolResponse := `{ "error": "mapper: validate policies: invalid destination protocol, specify either udp or tcp" }`

		DescribeTable("adding policies succeeds", addPoliciesSucceeds,
			Entry("v1", "v1", v1Request, v1Response),
			Entry("v0", "v0", v0Request, v0Response),
		)

		DescribeTable("failure cases", addPoliciesFails,
			Entry("v1: missing ports", "v1", v0Request, missingStartPortResponse),
			Entry("v1: missing protocol", "v1", v1RequestMissingProtocol, invalidProtocolResponse),

			Entry("v0: missing port", "v0", v1Request, missingPortResponse),
			Entry("v0: missing protocol", "v0", v0RequestMissingProtocol, invalidProtocolResponse),
		)
	})
})
