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

	"bytes"

	. "github.com/onsi/ginkgo"
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

	Describe("create and listing all destinations", func() {
		addPolicy := func(body string) {
			resp := helpers.MakeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/v1/external/policies", conf.ListenHost, conf.ListenPort),
				nil,
				strings.NewReader(body),
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(MatchJSON("{}"))
		}
		BeforeEach(func() {
			addPolicy(`{ "egress_policies": [ {"source": { "id": "live-app-1-guid" }, "destination": { "protocol": "tcp", "ips": [ {"start": "23.96.32.148", "end": "23.96.32.149" } ] } } ] }`)
			addPolicy(`{ "egress_policies": [ {"source": { "id": "live-app-1-guid" }, "destination": { "protocol": "tcp", "ports": [{"start": 8080, "end": 8081}], "ips": [ {"start": "23.96.32.150", "end": "23.96.32.151" } ] } } ] }`)
			addPolicy(`{ "egress_policies": [ {"source": { "id": "live-app-1-guid" }, "destination": { "protocol": "icmp", "icmp_type": 1, "icmp_code": 2, "ips": [ {"start": "23.96.32.150", "end": "23.96.32.151" } ] } } ] }`)
		})

		It("returns all created destinations", func() {

			createRequestBody := bytes.NewBufferString(`{
				"destinations": [	
					{
						"name": "my service",
						"description": "my service is a great service",	
						"ips": [{"start": "7211.30.35.9", "end": "72.30.35.9"}],
						"ports": [{"start": 8080, "end": 8080}],
						"protocol":"tcp"
					},
					{
						"name": "cloud infra",
						"description": "this is where my apps go",
						"ips": [{"start": "7211.30.35.9", "end": "72.30.35.9"}],
						"ports": [{"start": 8080, "end": 8080}],
						"protocol":"tcp"
					}
				]
			}`)
			resp := helpers.MakeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/v1/external/destinations", conf.ListenHost, conf.ListenPort),
				nil,
				createRequestBody,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusCreated))
			responseBytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(responseBytes)).To(MatchJSON(`{
				"total_destinations": 2,
				"destinations": [
					{
						"guid": "5",
						"name": "my service",
						"description": "my service is a great service",
						"ips": [{"start": "7211.30.35.9", "end": "72.30.35.9"}],
						"ports": [{"start": 8080, "end": 8080}],
						"protocol":"tcp"
					},
					{
						"guid": "6",
						"name": "cloud infra",
						"description": "this is where my apps go",
						"ips": [{"start": "7211.30.35.9", "end": "72.30.35.9"}],
						"ports": [{"start": 8080, "end": 8080}],
						"protocol":"tcp"
					}
				]
			}`))

			resp = helpers.MakeAndDoRequest(
				"GET",
				fmt.Sprintf("http://%s:%d/networking/v1/external/destinations", conf.ListenHost, conf.ListenPort),
				nil,
				nil,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseBytes, err = ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(responseBytes)).To(MatchJSON(`{
				"total_destinations": 5,
				"destinations": [
					{ "guid": "2", "protocol": "tcp", "ips": [ {"start": "23.96.32.148", "end": "23.96.32.149" } ] },
					{ "guid": "3", "protocol": "tcp", "ports": [{"start": 8080, "end": 8081}], "ips": [ {"start": "23.96.32.150", "end": "23.96.32.151" } ] },
					{ "guid": "4", "protocol": "icmp", "icmp_type": 1, "icmp_code": 2, "ips": [ {"start": "23.96.32.150", "end": "23.96.32.151" } ] },
					{ "guid": "5", "name": "my service", "description": "my service is a great service",	"ips": [{"start": "7211.30.35.9", "end": "72.30.35.9"}], "ports": [{"start": 8080, "end": 8080}], "protocol":"tcp" },
					{ "guid": "6", "name": "cloud infra", "description": "this is where my apps go", "ips": [{"start": "7211.30.35.9", "end": "72.30.35.9"}], "ports": [{"start": 8080, "end": 8080}], "protocol":"tcp" }
				]
			}`))

			Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
				HaveName("DestinationsIndexRequestTime"),
			))
		})
	})
})
