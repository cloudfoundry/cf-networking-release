package integration_test

import (
	"fmt"
	"github.com/nu7hatch/gouuid"
	"net/http"
	"policy-server/config"
	"policy-server/integration/helpers"
	"policy-server/psclient"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	"code.cloudfoundry.org/lager"
	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("External API Egress Policies", func() {
	var (
		sessions          []*gexec.Session
		conf              config.Config
		policyServerConfs []config.Config
		dbConf            db.Config
		client            *psclient.Client
		logger            lager.Logger

		fakeMetron metrics.FakeMetron
		token      string
	)

	BeforeEach(func() {
		fakeMetron = metrics.NewFakeMetron()

		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("external_api_create_test_node_%d", ports.PickAPort())

		template, _ := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
		policyServerConfs = configurePolicyServers(template, 2)
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]
		logger = lager.NewLogger("psclient")

		client = psclient.NewClient(logger, http.DefaultClient, fmt.Sprintf("http://%s:%d", conf.ListenHost, conf.ListenPort))

		token = "valid-token"
	})

	AfterEach(func() {
		stopPolicyServers(sessions, policyServerConfs)
		Expect(fakeMetron.Close()).To(Succeed())
	})

	Specify("a journey through egress policy", func() {
		someDest := psclient.Destination{
			Protocol: "tcp",
			IPs: []psclient.IPRange{
				{
					Start: "1.2.3.4",
					End:   "1.2.3.5",
				},
			},
			Ports: []psclient.Port{
				{
					Start: 8080,
					End:   9090,
				},
			},
		}

		unusedDest := psclient.Destination{
			Protocol: "udp",
			IPs: []psclient.IPRange{
				{
					Start: "3.2.3.4",
					End:   "3.2.3.5",
				},
			},
			Ports: []psclient.Port{
				{
					Start: 8082,
					End:   9092,
				},
			},
		}
		guids, err := client.CreateDestinations(token, someDest, unusedDest)
		Expect(err).NotTo(HaveOccurred())

		//TODO: assert the list returns the created destinations
		//destinations, err = client.ListDestinations(token, unusedDest)
		//Expect(e).NotTo(HaveOccurred())

		somePolicy := psclient.EgressPolicy{
			Source: psclient.EgressPolicySource{
				Type: "app",
				ID:   "some-app-guid",
			},
			Destination: psclient.EgressPolicyDestination{
				ID: guids[0],
			},
		}
		policyGUID, err := client.CreateEgressPolicy(somePolicy, token)
		Expect(err).NotTo(HaveOccurred())

		_, e := uuid.ParseHex(policyGUID)
		Expect(e).NotTo(HaveOccurred())

		//TODO: re-instate when index is an endpoint
		//egressPolicies, err := client.ListEgressPolicies(token)
		//Expect(err).NotTo(HaveOccurred())
		//Expect(egressPolicies).To(ConsistOf(somePolicy))
	})
})
