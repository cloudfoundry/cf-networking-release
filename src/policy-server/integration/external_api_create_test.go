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
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"

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
		dbConf.DatabaseName = fmt.Sprintf("external_api_create_test_node_%d", ports.PickAPort())

		template, _ := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
		policyServerConfs = configurePolicyServers(template, 2)
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]
	})

	AfterEach(func() {
		stopPolicyServers(sessions, policyServerConfs)

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Describe("adding app-to-app policies", func() {
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

	// Describe("adding egress policies", func() {
	// 	addPoliciesSucceeds := func(version, request, expectedResponse string) {
	// 		body := strings.NewReader(request)
	// 		resp := helpers.MakeAndDoRequest(
	// 			"POST",
	// 			fmt.Sprintf("http://%s:%d/networking/%s/external/policies", conf.ListenHost, conf.ListenPort, version),
	// 			nil,
	// 			body,
	// 		)

	// 		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	// 		responseString, err := ioutil.ReadAll(resp.Body)
	// 		Expect(err).NotTo(HaveOccurred())
	// 		Expect(responseString).To(MatchJSON("{}"))

	// 		resp = helpers.MakeAndDoRequest(
	// 			"GET",
	// 			fmt.Sprintf("http://%s:%d/networking/%s/external/policies", conf.ListenHost, conf.ListenPort, version),
	// 			nil,
	// 			nil,
	// 		)

	// 		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	// 		responseString, err = ioutil.ReadAll(resp.Body)
	// 		Expect(responseString).To(MatchJSON(expectedResponse))

	// 		Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
	// 			HaveName("CreatePoliciesRequestTime"),
	// 		))
	// 		Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
	// 			HaveName("StoreCreateWithTxSuccessTime"),
	// 		))
	// 		Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
	// 			HaveName("CollectionStoreCreateSuccessTime"),
	// 		))
	// 	}

	// 	v1Request := `{
	// 		"policies": [],
	// 		"egress_policies": [
	// 			{
	// 				"source": {
	// 					"id": "live-app-1-guid"
	// 				},
	// 				"destination": {
	// 					"ips": [{"start": "10.27.1.1", "end": "10.27.1.2"}],
	// 					"ports": [{"start": 8080, "end": 8081}],
	// 					"protocol": "tcp"
	// 				}
	// 			},
	// 			{
	// 				"source": {
	// 					"id": "live-app-1-guid",
	// 					"type": "app"
	// 				},
	// 				"destination": {
	// 					"ips": [{"start": "10.27.1.1", "end": "10.27.1.2"}],
	// 					"protocol": "icmp",
	// 					"icmp_type": 4,
	// 					"icmp_code": 3
	// 				}
	// 			}
	// 		]
	// 	}`

	// 	v1ExpectedResponse := `{
	// 		"total_policies": 0,
	// 		"policies": [],
	// 		"total_egress_policies": 2,
	// 		"egress_policies": [
	// 			{
	// 				"source": {
	// 					"id": "live-app-1-guid",
	// 					"type": "app"
	// 				},
	// 				"destination": {
	// 					"ips": [{"start": "10.27.1.1", "end": "10.27.1.2"}],
	// 					"ports": [{"start": 8080, "end": 8081}],
	// 					"protocol": "tcp"
	// 				}
	// 			},
	// 			{
	// 				"source": {
	// 					"id": "live-app-1-guid",
	// 					"type": "app"
	// 				},
	// 				"destination": {
	// 					"ips": [{"start": "10.27.1.1", "end": "10.27.1.2"}],
	// 					"protocol": "icmp",
	// 					"icmp_type": 4,
	// 					"icmp_code": 3
	// 				}
	// 			}
	// 		]
	// 	}`

	// 	v1RequestNoPorts := `{
	// 		"policies": [],
	// 		"egress_policies": [
	// 			{
	// 				"source": {
	// 					"id": "live-app-1-guid"
	// 				},
	// 				"destination": {
	// 					"ips": [{"start": "10.27.1.1", "end": "10.27.1.2"}],
	// 					"protocol": "tcp"
	// 				}
	// 			}
	// 		]
	// 	}`

	// 	v1ExpectedResponseNoPorts := `{
	// 		"total_policies": 0,
	// 		"policies": [],
	// 		"total_egress_policies": 1,
	// 		"egress_policies": [
	// 			{
	// 				"source": {
	// 					"id": "live-app-1-guid",
	// 					"type": "app"
	// 				},
	// 				"destination": {
	// 					"ips": [{"start": "10.27.1.1", "end": "10.27.1.2"}],
	// 					"protocol": "tcp"
	// 				}
	// 			}
	// 		]
	// 	}`

	// 	v1RequestSpaceEgress := `{
	// 		"policies": [],
	// 		"egress_policies": [
	// 			{
	// 				"source": {
	// 					"type": "space",
	// 					"id": "live-space-1-guid"
	// 				},
	// 				"destination": {
	// 					"ips": [{"start": "10.27.2.1", "end": "10.27.2.2"}],
	// 					"ports": [{"start": 8083, "end": 8086}],
	// 					"protocol": "udp"
	// 				}
	// 			}
	// 		]
	// 	}`

	// 	v1ExpectedResponseSpaceEgress := `{
	// 		"total_policies": 0,
	// 		"policies": [],
	// 		"total_egress_policies": 1,
	// 		"egress_policies": [
	// 			{
	// 				"source": {
	// 					"type": "space",
	// 					"id": "live-space-1-guid"
	// 				},
	// 				"destination": {
	// 					"ips": [{"start": "10.27.2.1", "end": "10.27.2.2"}],
	// 					"ports": [{"start": 8083, "end": 8086}],
	// 					"protocol": "udp"
	// 				}
	// 			}
	// 		]
	// 	}`

	// 	DescribeTable("adding policies succeeds", addPoliciesSucceeds,
	// 		Entry("v1", "v1", v1Request, v1ExpectedResponse),
	// 		Entry("v1 no ports", "v1", v1RequestNoPorts, v1ExpectedResponseNoPorts),
	// 		Entry("v1 space egress", "v1", v1RequestSpaceEgress, v1ExpectedResponseSpaceEgress),
	// 	)
	// })
})
