package integration_test

import (
	"fmt"
	"net/http"
	"policy-server/config"
	"policy-server/integration/helpers"
	"policy-server/psclient"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	uuid "github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
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
		logger = lagertest.NewTestLogger("psclient")

		client = psclient.NewClient(logger, http.DefaultClient, fmt.Sprintf("http://%s:%d", conf.ListenHost, conf.ListenPort))

		token = "valid-token"
	})

	AfterEach(func() {
		stopPolicyServers(sessions, policyServerConfs)
		Expect(fakeMetron.Close()).To(Succeed())
	})

	Specify("a journey through egress policy", func() {
		someDest := psclient.Destination{
			Name:        "tcp with ports",
			Description: "dest description",
			Protocol:    "tcp",
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

		anotherDest := psclient.Destination{
			Name:        "udp destination",
			Description: "another description",
			Protocol:    "udp",
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
		createdDestinations, err := client.CreateDestinations(token, someDest, anotherDest)
		Expect(err).NotTo(HaveOccurred())

		for _, createdDestination := range createdDestinations {
			_, err = uuid.ParseHex(createdDestination.GUID)
			Expect(err).NotTo(HaveOccurred())
		}

		somePolicy := psclient.EgressPolicy{
			Source: psclient.EgressPolicySource{
				Type: "app",
				ID:   "live-app-1-guid",
			},
			Destination: psclient.Destination{
				GUID: createdDestinations[0].GUID,
			},
			AppLifecycle: "running",
		}
		policyGUID, err := client.CreateEgressPolicy(somePolicy, token)
		Expect(err).NotTo(HaveOccurred())

		_, err = uuid.ParseHex(policyGUID)
		Expect(err).NotTo(HaveOccurred())
		egressPolicyList, err := client.ListEgressPolicies(token, []string{}, []string{}, []string{}, []string{})
		Expect(err).NotTo(HaveOccurred())
		egressPolicies := egressPolicyList.EgressPolicies
		Expect(egressPolicies).To(HaveLen(1))
		Expect(egressPolicies[0]).To(Equal(
			psclient.EgressPolicy{
				GUID: policyGUID,
				Source: psclient.EgressPolicySource{
					ID:   "live-app-1-guid",
					Type: "app",
				},
				Destination: psclient.Destination{
					GUID:        createdDestinations[0].GUID,
					Name:        "tcp with ports",
					Description: "dest description",
					Protocol:    "tcp",
					IPs:         []psclient.IPRange{{Start: "1.2.3.4", End: "1.2.3.5"}},
					Ports:       []psclient.Port{{Start: 8080, End: 9090}},
				},
				AppLifecycle: "running",
			},
		))

		//check idempotent create
		_, err = client.CreateEgressPolicy(somePolicy, token)
		Expect(err).NotTo(HaveOccurred())
		egressPolicyList, err = client.ListEgressPolicies(token, []string{}, []string{}, []string{}, []string{})
		Expect(err).NotTo(HaveOccurred())
		egressPolicies = egressPolicyList.EgressPolicies
		Expect(egressPolicies).To(HaveLen(1))

		someSecondPolicy := psclient.EgressPolicy{
			Source: psclient.EgressPolicySource{
				Type: "app",
				ID:   "live-app-2-guid",
			},
			Destination: psclient.Destination{
				GUID: createdDestinations[1].GUID,
			},
			AppLifecycle: "staging",
		}

		secondPolicyGUID, err := client.CreateEgressPolicy(someSecondPolicy, token)
		Expect(err).NotTo(HaveOccurred())

		_, err = uuid.ParseHex(secondPolicyGUID)
		Expect(err).NotTo(HaveOccurred())

		someThirdPolicy := psclient.EgressPolicy{
			Source: psclient.EgressPolicySource{
				Type: "app",
				ID:   "live-app-3-guid",
			},
			Destination: psclient.Destination{
				GUID: createdDestinations[1].GUID,
			},
			AppLifecycle: "all",
		}

		thirdPolicyGUID, err := client.CreateEgressPolicy(someThirdPolicy, token)
		Expect(err).NotTo(HaveOccurred())

		_, err = uuid.ParseHex(thirdPolicyGUID)
		Expect(err).NotTo(HaveOccurred())

		By("fetching list of IDs")
		egressPolicyList, err = client.ListEgressPolicies(token, []string{"live-app-1-guid", "live-app-3-guid"}, []string{"app"}, []string{}, []string{})
		Expect(err).NotTo(HaveOccurred())
		egressPolicies = egressPolicyList.EgressPolicies
		Expect(egressPolicies).To(HaveLen(2))
		Expect(egressPolicies[0]).To(Equal(
			psclient.EgressPolicy{
				GUID: policyGUID,
				Source: psclient.EgressPolicySource{
					ID:   "live-app-1-guid",
					Type: "app",
				},
				Destination: psclient.Destination{
					GUID:        createdDestinations[0].GUID,
					Name:        "tcp with ports",
					Description: "dest description",
					Protocol:    "tcp",
					IPs:         []psclient.IPRange{{Start: "1.2.3.4", End: "1.2.3.5"}},
					Ports:       []psclient.Port{{Start: 8080, End: 9090}},
				},
				AppLifecycle: "running",
			},
		))
		Expect(egressPolicies[1]).To(Equal(
			psclient.EgressPolicy{
				GUID: thirdPolicyGUID,
				Source: psclient.EgressPolicySource{
					ID:   "live-app-3-guid",
					Type: "app",
				},
				Destination: psclient.Destination{
					GUID:        createdDestinations[1].GUID,
					Name:        "udp destination",
					Description: "another description",
					Protocol:    "udp",
					IPs:         []psclient.IPRange{{Start: "3.2.3.4", End: "3.2.3.5"}},
					Ports:       []psclient.Port{{Start: 8082, End: 9092}},
				},
				AppLifecycle: "all",
			},
		))

		By("ANDing search filter params")
		egressPolicyList, err = client.ListEgressPolicies(token, []string{"live-app-1-guid", "live-app-3-guid"}, []string{"app"}, []string{createdDestinations[0].GUID}, []string{})
		Expect(err).NotTo(HaveOccurred())
		egressPolicies = egressPolicyList.EgressPolicies
		Expect(egressPolicies).To(HaveLen(1))
		Expect(egressPolicies[0]).To(Equal(
			psclient.EgressPolicy{
				GUID: policyGUID,
				Source: psclient.EgressPolicySource{
					ID:   "live-app-1-guid",
					Type: "app",
				},
				Destination: psclient.Destination{
					GUID:        createdDestinations[0].GUID,
					Name:        "tcp with ports",
					Description: "dest description",
					Protocol:    "tcp",
					IPs:         []psclient.IPRange{{Start: "1.2.3.4", End: "1.2.3.5"}},
					Ports:       []psclient.Port{{Start: 8080, End: 9090}},
				},
				AppLifecycle: "running",
			},
		))

		deletedDestinations, err := client.DeleteDestination(token, createdDestinations[0])
		Expect(err).To(HaveOccurred(), "expected the delete to fail because this destinations still has associated egress policy")
		Expect(err).To(MatchError(ContainSubstring("destination is still in use")))

		deletedEgressPolicy, err := client.DeleteEgressPolicy(policyGUID, token)
		Expect(err).NotTo(HaveOccurred())
		somePolicy.GUID = policyGUID
		Expect(somePolicy).To(Equal(deletedEgressPolicy))

		deletedEgressPolicy, err = client.DeleteEgressPolicy(secondPolicyGUID, token)
		Expect(err).NotTo(HaveOccurred())
		someSecondPolicy.GUID = secondPolicyGUID
		Expect(someSecondPolicy).To(Equal(deletedEgressPolicy))

		deletedEgressPolicy, err = client.DeleteEgressPolicy(thirdPolicyGUID, token)
		Expect(err).NotTo(HaveOccurred())
		someThirdPolicy.GUID = thirdPolicyGUID
		Expect(someThirdPolicy).To(Equal(deletedEgressPolicy))

		egressPolicyList, err = client.ListEgressPolicies(token, []string{}, []string{}, []string{}, []string{})
		Expect(err).NotTo(HaveOccurred())
		egressPolicies = egressPolicyList.EgressPolicies
		Expect(egressPolicies).To(HaveLen(0))

		deletedDestinations, err = client.DeleteDestination(token, createdDestinations[0])
		Expect(err).NotTo(HaveOccurred())
		Expect(deletedDestinations).To(HaveLen(1))
		Expect(deletedDestinations[0]).To(Equal(createdDestinations[0]))

		By("deleting a destination that does not exist")
		deletedDestinations, err = client.DeleteDestination(token, createdDestinations[0])
		Expect(err).NotTo(HaveOccurred())
		Expect(deletedDestinations).To(HaveLen(0))
	})
})
