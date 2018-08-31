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

var _ = Describe("External Destination API", func() {
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
		dbConf.DatabaseName = fmt.Sprintf("external_api_destination_test_node_%d", ports.PickAPort())

		template, _ := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
		policyServerConfs = configurePolicyServers(template, 2)
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]
	})

	AfterEach(func() {
		stopPolicyServers(sessions, policyServerConfs)

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Describe("listing destinations", func() {
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
			addPolicy("v1", `{ "egress_policies": [ {"source": { "id": "live-app-1-guid" }, "destination": { "protocol": "tcp", "ips": [ {"start": "23.96.32.148", "end": "23.96.32.149" } ] } } ] }`)
			addPolicy("v1", `{ "egress_policies": [ {"source": { "id": "live-app-1-guid" }, "destination": { "protocol": "tcp", "ports": [{"start": 8080, "end": 8081}], "ips": [ {"start": "23.96.32.150", "end": "23.96.32.151" } ] } } ] }`)
			addPolicy("v1", `{ "egress_policies": [ {"source": { "id": "live-app-1-guid" }, "destination": { "protocol": "icmp", "icmp_type": 1, "icmp_code": 2, "ips": [ {"start": "23.96.32.150", "end": "23.96.32.151" } ] } } ] }`)
		})

		listDestinations := func(version, queryString, expectedResponse string) {
			resp := helpers.MakeAndDoRequest(
				"GET",
				fmt.Sprintf("http://%s:%d/networking/%s/external/destinations%s", conf.ListenHost, conf.ListenPort, version, queryString),
				nil,
				nil,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseBytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(responseBytes)).To(MatchJSON(expectedResponse))

			//move this to it's own it block
			Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
				HaveName("DestinationsIndexRequestTime"),
			))
		}

		v1Response := `{
			"total_destinations": 3,
			"destinations": [
				{ "guid": "2", "name": " ", "description": " ", "protocol": "tcp", "ips": [ {"start": "23.96.32.148", "end": "23.96.32.149" } ] },
				{ "guid": "3", "name": " ", "description": " ", "protocol": "tcp", "ports": [{"start": 8080, "end": 8081}], "ips": [ {"start": "23.96.32.150", "end": "23.96.32.151" } ] },
				{ "guid": "4", "name": " ", "description": " ", "protocol": "icmp", "icmp_type": 1, "icmp_code": 2, "ips": [ {"start": "23.96.32.150", "end": "23.96.32.151" } ] }
			]
		}`

		DescribeTable("listing all destinations", listDestinations,
			Entry("v1: all", "v1", "", v1Response),
		)
	})
})
