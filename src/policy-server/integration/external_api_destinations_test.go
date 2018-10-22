package integration_test

import (
	"fmt"
	"net/http"
	"policy-server/config"
	"policy-server/integration/helpers"
	"policy-server/psclient"
	"regexp"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/nu7hatch/gouuid"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("External Destination API", func() {
	var (
		sessions          []*gexec.Session
		policyServerConfs []config.Config
		dbConf            db.Config
		client            *psclient.Client
		logger            lager.Logger
		fakeMetron        metrics.FakeMetron
		token             string
	)

	BeforeEach(func() {
		fakeMetron = metrics.NewFakeMetron()

		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("external_api_destination_test_node_%d", ports.PickAPort())

		template, _ := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
		policyServerConfs = configurePolicyServers(template, 2)
		sessions = startPolicyServers(policyServerConfs)

		conf := policyServerConfs[0]

		token = "valid-token"
		logger = lagertest.NewTestLogger("psclient")
		client = psclient.NewClient(logger, http.DefaultClient, fmt.Sprintf("http://%s:%d", conf.ListenHost, conf.ListenPort))
	})

	AfterEach(func() {
		stopPolicyServers(sessions, policyServerConfs)

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Describe("create and listing all destinations", func() {
		It("returns all created destinations", func() {
			By("checking that invalid requests result in 400 error code response")
			_, err := client.CreateDestinations(token)
			Expect(err).To(MatchError(MatchRegexp("http status 400.*missing destinations")))

			By("testing a happy-path journey")
			icmpType := 1
			icmpCode := 2

			toBeCreated := []psclient.Destination{
				{
					Name:        "tcp ips only",
					Description: "tcp ips only desc",
					Ports:       []psclient.Port{{Start: 8080, End: 8081}},
					IPs:         []psclient.IPRange{{Start: "23.96.32.148", End: "23.96.32.149"}},
					Protocol:    "tcp",
				},
				{
					Name:        "udp ips and ports",
					Description: "udp ips and ports desc",
					Protocol:    "udp",
					Ports:       []psclient.Port{{Start: 8080, End: 8081}},
					IPs:         []psclient.IPRange{{Start: "23.96.32.150", End: "23.96.32.151"}},
				},
				{
					Name:        "icmp with type code",
					Description: "icmp with type code",
					ICMPType:    &icmpType,
					ICMPCode:    &icmpCode,
					IPs:         []psclient.IPRange{{Start: "23.96.32.150", End: "23.96.32.151"}},
					Protocol:    "icmp",
				},
			}

			createdDestinations, err := client.CreateDestinations(token, toBeCreated...)
			Expect(err).NotTo(HaveOccurred())

			Expect(createdDestinations).To(HaveLen(3))

			_, err = uuid.ParseHex(createdDestinations[0].GUID)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdDestinations[0].Name).To(Equal("tcp ips only"))
			Expect(createdDestinations[0].Description).To(Equal("tcp ips only desc"))
			Expect(createdDestinations[0].Ports).To(Equal([]psclient.Port{{Start: 8080, End: 8081}}))
			Expect(createdDestinations[0].IPs).To(Equal([]psclient.IPRange{{Start: "23.96.32.148", End: "23.96.32.149"}}))
			Expect(createdDestinations[0].Protocol).To(Equal("tcp"))

			_, err = uuid.ParseHex(createdDestinations[1].GUID)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdDestinations[1].Name).To(Equal("udp ips and ports"))
			Expect(createdDestinations[1].Description).To(Equal("udp ips and ports desc"))
			Expect(createdDestinations[1].Ports).To(Equal([]psclient.Port{{Start: 8080, End: 8081}}))
			Expect(createdDestinations[1].IPs).To(Equal([]psclient.IPRange{{Start: "23.96.32.150", End: "23.96.32.151"}}))
			Expect(createdDestinations[1].Protocol).To(Equal("udp"))

			_, err = uuid.ParseHex(createdDestinations[2].GUID)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdDestinations[2].Name).To(Equal("icmp with type code"))
			Expect(createdDestinations[2].Description).To(Equal("icmp with type code"))
			Expect(createdDestinations[2].IPs).To(Equal([]psclient.IPRange{{Start: "23.96.32.150", End: "23.96.32.151"}}))
			Expect(createdDestinations[2].Protocol).To(Equal("icmp"))
			Expect(createdDestinations[2].ICMPCode).To(Equal(&icmpCode))
			Expect(createdDestinations[2].ICMPType).To(Equal(&icmpType))

			By("listing the existing destinations")
			listedDestinations, err := client.ListDestinations(token, psclient.ListDestinationsOptions{})
			Expect(err).NotTo(HaveOccurred())

			Expect(listedDestinations).To(Equal(createdDestinations))

			By("listing destinations with name filter")
			listedDestinations, err = client.ListDestinations(token, psclient.ListDestinationsOptions{
				QueryNames: []string{createdDestinations[1].Name, createdDestinations[2].Name},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(listedDestinations).To(HaveLen(2))
			Expect(listedDestinations).To(ConsistOf(createdDestinations[1], createdDestinations[2]))

			By("listing destinations with guid filter")
			listedDestinations, err = client.ListDestinations(token, psclient.ListDestinationsOptions{
				QueryIDs: []string{createdDestinations[0].GUID, createdDestinations[2].GUID},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(listedDestinations).To(HaveLen(2))
			Expect(listedDestinations).To(ConsistOf(createdDestinations[0], createdDestinations[2]))

			By("attempting to duplicate destinations")
			_, err = client.CreateDestinations(token, toBeCreated...)
			Expect(err).To(MatchError(MatchRegexp("http status 400.*entry with name 'tcp ips only' already exists")))

			By("updating one of the destinations")
			destToUpdate := createdDestinations[1]
			destToUpdate.Name = "new name"
			destToUpdate.Ports[0].End = 8080
			updatedDests, err := client.UpdateDestinations(token, destToUpdate)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedDests).To(HaveLen(1))
			Expect(updatedDests[0]).To(Equal(destToUpdate))

			By("listing all destinations and confirming that the update was persisted")
			listedDestinations, err = client.ListDestinations(token, psclient.ListDestinationsOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(listedDestinations).To(ConsistOf(createdDestinations[0], updatedDests[0], createdDestinations[2]))

			Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
				HaveName("DestinationsIndexRequestTime"),
			))
		})
	})
})

var replaceGUIDRegex = regexp.MustCompile(`"id":"[a-z0-9\-]{36}"`)

func replaceGUID(value string) string {
	return string(replaceGUIDRegex.ReplaceAll([]byte(value), []byte(`"id":"<replaced>"`)))
}
