package integration_test

import (
	"encoding/json"
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

type policiesResponse struct {
	TotalPolicies int                      `json:"total_policies"`
	Policies      []map[string]interface{} `json:"policies"`
}

var _ = Describe("External API Listing Policies", func() {
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
		dbConf.DatabaseName = fmt.Sprintf("external_api_index_test_node_%d", ports.PickAPort())

		template, _ := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
		policyServerConfs = configurePolicyServers(template, 2)
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]
	})

	AfterEach(func() {
		stopPolicyServers(sessions, policyServerConfs)

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Describe("listing policies", func() {
		addPolicy := func(version, body string) {
			resp := helpers.MakeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/%s/external/policies", conf.ListenHost, conf.ListenPort, version),
				nil,
				strings.NewReader(body),
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(MatchJSON("{}"))
		}
		BeforeEach(func() {
			addPolicy("v1", `{ "policies": [ {"source": { "id": "app1" }, "destination": { "id": "app2", "protocol": "tcp", "ports": { "start": 1234, "end": 1234 } } } ] }`)
			addPolicy("v1", `{ "policies": [ {"source": { "id": "app3" }, "destination": { "id": "app1", "protocol": "tcp", "ports": { "start": 8080, "end": 8090 } } } ] }`)
			addPolicy("v0", `{ "policies": [ {"source": { "id": "app3" }, "destination": { "id": "app4", "protocol": "tcp", "port": 7777 } } ] }`)
		})

		listPolicies := func(version, queryString, expectedResponse string) {
			var expectedResponseJson policiesResponse
			err := json.Unmarshal([]byte(expectedResponse), &expectedResponseJson)
			Expect(err).NotTo(HaveOccurred())

			resp := helpers.MakeAndDoRequest(
				"GET",
				fmt.Sprintf("http://%s:%d/networking/%s/external/policies%s", conf.ListenHost, conf.ListenPort, version, queryString),
				nil,
				nil,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			var responseJson policiesResponse
			err = json.Unmarshal(responseString, &responseJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseJson.TotalPolicies).To(Equal(expectedResponseJson.TotalPolicies))
			Expect(responseJson.Policies).To(ConsistOf(expectedResponseJson.Policies))

			Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
				HaveName("PoliciesIndexRequestTime"),
			))
		}

		v1Response := `{ "total_policies": 3, "policies": [
				{ "source": { "id": "app1" }, "destination": { "id": "app2", "protocol": "tcp", "ports": { "start": 1234, "end": 1234 } } },
				{ "source": { "id": "app3" }, "destination": { "id": "app1", "protocol": "tcp", "ports": { "start": 8080, "end": 8090 } } },
				{ "source": { "id": "app3" }, "destination": { "id": "app4", "protocol": "tcp", "ports": { "start": 7777, "end": 7777 } } }
			]
		}`
		v1ResponseFiltered := `{ "total_policies": 2, "policies": [
			{ "source": { "id": "app1" }, "destination": { "id": "app2", "protocol": "tcp", "ports": { "start": 1234, "end": 1234 } } },
			{ "source": { "id": "app3" }, "destination": { "id": "app1", "protocol": "tcp", "ports": { "start": 8080, "end": 8090 } } }
		]}`
		v1ResponseSourceFiltered := `{ "total_policies": 1, "policies": [
			{ "source": { "id": "app1" }, "destination": { "id": "app2", "protocol": "tcp", "ports": { "start": 1234, "end": 1234 } } }
		]}`
		v1ResponseDestFiltered := `{ "total_policies": 2, "policies": [
			{ "source": { "id": "app3" }, "destination": { "id": "app4", "protocol": "tcp", "ports": { "start": 7777, "end": 7777 } } },
			{ "source": { "id": "app3" }, "destination": { "id": "app1", "protocol": "tcp", "ports": { "start": 8080, "end": 8090 } } }
		]}`
		v1ResponseSourceAndDestFiltered := `{ "total_policies": 2, "policies": [
			{ "source": { "id": "app3" }, "destination": { "id": "app4", "protocol": "tcp", "ports": { "start": 7777, "end": 7777 } } },
			{ "source": { "id": "app1" }, "destination": { "id": "app2", "protocol": "tcp", "ports": { "start": 1234, "end": 1234 } } }
		]}`

		v0Response := `{ "total_policies": 2, "policies": [
			{ "source": { "id": "app1" }, "destination": { "id": "app2", "protocol": "tcp", "port": 1234 } },
			{ "source": { "id": "app3" }, "destination": { "id": "app4", "protocol": "tcp", "port": 7777 } }
		]}`
		v0ResponseFiltered := `{ "total_policies": 1, "policies": [
			{ "source": { "id": "app1" }, "destination": { "id": "app2", "protocol": "tcp", "port": 1234 } }
		]}`
		v0ResponseSourceFiltered := `{ "total_policies": 1, "policies": [
			{ "source": { "id": "app1" }, "destination": { "id": "app2", "protocol": "tcp", "port": 1234 } }
		]}`
		v0ResponseDestFiltered := `{ "total_policies": 1, "policies": [
			{ "source": { "id": "app3" }, "destination": { "id": "app4", "protocol": "tcp", "port": 7777 } }
		]}`
		v0ResponseSourceAndDestFiltered := `{ "total_policies": 2, "policies": [
			{ "source": { "id": "app3" }, "destination": { "id": "app4", "protocol": "tcp", "port": 7777 } },
			{ "source": { "id": "app1" }, "destination": { "id": "app2", "protocol": "tcp", "port": 1234 } }
		]}`

		DescribeTable("listing all policies", listPolicies,
			Entry("v1: all", "v1", "", v1Response),
			Entry("v0: all", "v0", "", v0Response),
		)

		DescribeTable("listing policies filtered", listPolicies,
			Entry("v1: id filtered", "v1", "?id=app1,app2", v1ResponseFiltered),
			Entry("v0: id filtered", "v0", "?id=app1,app2", v0ResponseFiltered),
			Entry("v1: source_id filtered", "v1", "?source_id=app1,app2", v1ResponseSourceFiltered),
			Entry("v0: source_id filtered", "v0", "?source_id=app1,app2", v0ResponseSourceFiltered),
			Entry("v1: dest_id filtered", "v1", "?dest_id=app1,app4", v1ResponseDestFiltered),
			Entry("v0: dest_id filtered", "v0", "?dest_id=app1,app4", v0ResponseDestFiltered),
			Entry("v1: source_id and dest_id filtered", "v1", "?source_id=app1,app3&dest_id=app2,app4", v1ResponseSourceAndDestFiltered),
			Entry("v0: source_id and dest_id filtered", "v0", "?source_id=app1,app3&dest_id=app2,app4", v0ResponseSourceAndDestFiltered),
		)
	})
})
