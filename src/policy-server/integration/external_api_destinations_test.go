package integration_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"policy-server/config"
	"policy-server/integration/helpers"
	"regexp"

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
		destinationsURL   string
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

		conf := policyServerConfs[0]
		destinationsURL = fmt.Sprintf("http://%s:%d/networking/v1/external/destinations", conf.ListenHost, conf.ListenPort)
	})

	AfterEach(func() {
		stopPolicyServers(sessions, policyServerConfs)

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Describe("create and listing all destinations", func() {
		It("returns all created destinations", func() {
			invalidCreateRequestBody := bytes.NewBufferString(`{
				"destinations": []
			}`)

			resp := helpers.MakeAndDoRequest("POST", destinationsURL, nil, invalidCreateRequestBody)
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

			By("creating the initial destinations")
			getCreateRequestBody := func() *bytes.Buffer {
				return bytes.NewBufferString(`{
				"destinations": [	
					{
						"name": "tcp ips only",
						"description": "tcp ips only desc",
						"ports": [{"start": 8080, "end": 8081}],
						"ips": [{"start": "23.96.32.148", "end": "23.96.32.149" }],
						"protocol": "tcp"
					},
					{
						"name": "udp ips and ports",
						"description": "udp ips and ports desc",
						"protocol": "udp",
						"ports": [{"start": 8080, "end": 8081}],
						"ips": [{"start": "23.96.32.150", "end": "23.96.32.151"}]
					},
					{
						"name": "icmp with type code",
						"description": "icmp with type code",
						"icmp_type": 1,
						"icmp_code": 2,
						"ips": [{"start": "23.96.32.150", "end": "23.96.32.151"}],
						"protocol": "icmp"
					}
				]
			}`)
			}
			resp = helpers.MakeAndDoRequest("POST", destinationsURL, nil, getCreateRequestBody())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))
			responseBytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(responseBytes)).To(WithTransform(replaceGUID, MatchJSON(`{
				"total_destinations": 3,
				"destinations": [
					{
						"id": "<replaced>",
						"name": "tcp ips only",
						"description": "tcp ips only desc",
						"ports": [{"start": 8080, "end": 8081}],
						"ips": [{"start": "23.96.32.148", "end": "23.96.32.149" }],
						"protocol": "tcp"
					},
					{
						"id": "<replaced>",
						"name": "udp ips and ports",
						"description": "udp ips and ports desc",
						"protocol": "udp",
						"ports": [{"start": 8080, "end": 8081}],
						"ips": [{"start": "23.96.32.150", "end": "23.96.32.151"}]
					},
					{
						"id": "<replaced>",
						"name": "icmp with type code",
						"description": "icmp with type code",
						"icmp_type": 1,
						"icmp_code": 2,
						"ips": [{"start": "23.96.32.150", "end": "23.96.32.151"}],
						"protocol": "icmp"
					}
				]
			}`)))

			By("listing the existing destinations")
			resp = helpers.MakeAndDoRequest("GET", destinationsURL, nil, nil)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseBytes, err = ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(responseBytes)).To(WithTransform(replaceGUID, MatchJSON(`{
				"total_destinations": 3,
				"destinations": [
					{
						"id": "<replaced>",
						"name": "tcp ips only",
						"description": "tcp ips only desc",
						"ports": [{"start": 8080, "end": 8081}],
						"ips": [{"start": "23.96.32.148", "end": "23.96.32.149" }],
						"protocol": "tcp"
					},
					{
						"id": "<replaced>",
						"name": "udp ips and ports",
						"description": "udp ips and ports desc",
						"protocol": "udp",
						"ports": [{"start": 8080, "end": 8081}],
						"ips": [{"start": "23.96.32.150", "end": "23.96.32.151"}]
					},
					{
						"id": "<replaced>",
						"name": "icmp with type code",
						"description": "icmp with type code",
						"icmp_type": 1,
						"icmp_code": 2,
						"ips": [{"start": "23.96.32.150", "end": "23.96.32.151"}],
						"protocol": "icmp"
					}
				]
			}`)))

			By("attempting to duplicate destinations")
			resp = helpers.MakeAndDoRequest("POST", destinationsURL, nil, getCreateRequestBody())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			responseBytes, err = ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(responseBytes)).To(ContainSubstring("duplicate name error"))

			Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
				HaveName("DestinationsIndexRequestTime"),
			))
		})
	})
})

var replaceGUIDRegex = regexp.MustCompile(`"id":"[^"]*"`)

func replaceGUID(value string) string {
	return string(replaceGUIDRegex.ReplaceAll([]byte(value), []byte(`"id":"<replaced>"`)))
}
