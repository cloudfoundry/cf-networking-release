package integration_test

import (
	"fmt"
	"net/http"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/policy-server/config"
	"code.cloudfoundry.org/policy-server/integration/helpers"
	"code.cloudfoundry.org/policy-server/psclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const (
	GUID_REGEX = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
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

		tcpDest             psclient.Destination
		udpDest             psclient.Destination
		allDest             psclient.Destination
		createdDestinations []psclient.Destination

		somePolicy             psclient.EgressPolicy
		someSecondPolicy       psclient.EgressPolicy
		someThirdPolicy        psclient.EgressPolicy
		policyWithAllProtocols psclient.EgressPolicy

		expectedSomePolicy             psclient.EgressPolicy
		expectedSomeSecondPolicy       psclient.EgressPolicy
		expectedSomeThirdPolicy        psclient.EgressPolicy
		expectedPolicyWithAllProtocols psclient.EgressPolicy
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

		tcpDest = psclient.Destination{
			Name:        "tcp with ports",
			Description: "dest description",
			Rules: []psclient.DestinationRule{
				{
					Description: "dis is cool",
					Protocol:    "tcp",
					IPs:         "1.2.3.4-1.2.3.5",
					Ports:       "8080-9090",
				},
			},
		}

		udpDest = psclient.Destination{
			Name:        "udp destination",
			Description: "another description",
			Rules: []psclient.DestinationRule{
				{
					Protocol: "udp",
					IPs:      "3.2.3.4-3.2.3.5",
					Ports:    "8082-9092",
				},
			},
		}

		allDest = psclient.Destination{
			Name:        "all protocol destination",
			Description: "description",
			Rules: []psclient.DestinationRule{
				{
					Description: "meow rule description",
					Protocol:    "all",
					IPs:         "3.2.3.4-3.2.3.5",
					Ports:       "8082-9092",
				},
			},
		}

		By("setting up destinations")
		var err error
		createdDestinations, err = client.CreateDestinations(token, tcpDest, udpDest, allDest)
		Expect(err).NotTo(HaveOccurred())

		somePolicy = psclient.EgressPolicy{
			Source: psclient.EgressPolicySource{
				Type: "app",
				ID:   "live-app-1-guid",
			},
			Destination: psclient.Destination{
				GUID: createdDestinations[0].GUID,
			},
			AppLifecycle: "running",
		}

		someSecondPolicy = psclient.EgressPolicy{
			Source: psclient.EgressPolicySource{
				Type: "default",
			},
			Destination: psclient.Destination{
				GUID: createdDestinations[1].GUID,
			},
			AppLifecycle: "staging",
		}

		someThirdPolicy = psclient.EgressPolicy{
			Source: psclient.EgressPolicySource{
				Type: "app",
				ID:   "live-app-3-guid",
			},
			Destination: psclient.Destination{
				GUID: createdDestinations[1].GUID,
			},
			AppLifecycle: "all",
		}

		expectedSomePolicy = psclient.EgressPolicy{
			Source: psclient.EgressPolicySource{
				ID:   "live-app-1-guid",
				Type: "app",
			},
			Destination:  createdDestinations[0],
			AppLifecycle: "running",
		}

		expectedSomeSecondPolicy = psclient.EgressPolicy{
			Source: psclient.EgressPolicySource{
				Type: "default",
			},
			Destination:  createdDestinations[1],
			AppLifecycle: "staging",
		}

		expectedSomeThirdPolicy = psclient.EgressPolicy{
			Source: psclient.EgressPolicySource{
				Type: "app",
				ID:   "live-app-3-guid",
			},
			Destination:  createdDestinations[1],
			AppLifecycle: "all",
		}

		policyWithAllProtocols = psclient.EgressPolicy{
			Source: psclient.EgressPolicySource{
				Type: "app",
				ID:   "live-app-1-guid",
			},
			Destination: psclient.Destination{
				GUID: createdDestinations[2].GUID,
			},
			AppLifecycle: "running",
		}

		expectedPolicyWithAllProtocols = psclient.EgressPolicy{
			Source: psclient.EgressPolicySource{
				Type: "app",
				ID:   "live-app-1-guid",
			},
			Destination:  createdDestinations[2],
			AppLifecycle: "running",
		}
	})

	AfterEach(func() {
		deletedDestinations, err := client.DeleteDestination(token, createdDestinations[0])
		Expect(err).NotTo(HaveOccurred())
		Expect(deletedDestinations).To(HaveLen(1))
		Expect(deletedDestinations[0]).To(Equal(createdDestinations[0]))

		By("deleting a destination that does not exist")
		deletedDestinations, err = client.DeleteDestination(token, createdDestinations[0])
		Expect(err).NotTo(HaveOccurred())
		Expect(deletedDestinations).To(HaveLen(0))

		stopPolicyServers(sessions, policyServerConfs)
		Expect(fakeMetron.Close()).To(Succeed())
	})

	Specify("a journey through egress policy", func() {
		By("setting up policies")
		policyGUID, err := client.CreateEgressPolicy(somePolicy, token)
		Expect(err).NotTo(HaveOccurred())
		Expect(policyGUID).To(MatchRegexp(GUID_REGEX))

		egressPolicyList, err := client.ListEgressPolicies(token, []string{}, []string{}, []string{}, []string{})
		Expect(err).NotTo(HaveOccurred())
		Expect(egressPolicyList.EgressPolicies).To(HaveLen(1))
		Expect(egressPolicyList.EgressPolicies).To(WithTransform(replaceEgressPoliciesGUID, ConsistOf(expectedSomePolicy)))

		By("checking idempotent create")
		_, err = client.CreateEgressPolicy(somePolicy, token)
		Expect(err).NotTo(HaveOccurred())

		egressPolicyList, err = client.ListEgressPolicies(token, []string{}, []string{}, []string{}, []string{})
		Expect(err).NotTo(HaveOccurred())
		egressPolicies := egressPolicyList.EgressPolicies
		Expect(egressPolicies).To(HaveLen(1))

		By("adding more policies")
		secondPolicyGUID, err := client.CreateEgressPolicy(someSecondPolicy, token)
		Expect(err).NotTo(HaveOccurred())
		Expect(secondPolicyGUID).To(MatchRegexp(GUID_REGEX))

		thirdPolicyGUID, err := client.CreateEgressPolicy(someThirdPolicy, token)
		Expect(err).NotTo(HaveOccurred())
		Expect(thirdPolicyGUID).To(MatchRegexp(GUID_REGEX))

		policyWithAllProtocolsGUID, err := client.CreateEgressPolicy(policyWithAllProtocols, token)
		Expect(err).NotTo(HaveOccurred())
		Expect(policyWithAllProtocolsGUID).To(MatchRegexp(GUID_REGEX))

		By("listing all policies")
		egressPolicyList, err = client.ListEgressPolicies(token, []string{}, []string{}, []string{}, []string{})
		Expect(err).NotTo(HaveOccurred())
		egressPolicies = egressPolicyList.EgressPolicies
		Expect(egressPolicies).To(HaveLen(4))
		Expect(egressPolicies).To(WithTransform(replaceEgressPoliciesGUID, ConsistOf(expectedSomePolicy, expectedSomeSecondPolicy, expectedSomeThirdPolicy, expectedPolicyWithAllProtocols)))

		By("fetching list of IDs")
		egressPolicyList, err = client.ListEgressPolicies(token, []string{"live-app-1-guid", "live-app-3-guid"}, []string{"app"}, []string{}, []string{})
		Expect(err).NotTo(HaveOccurred())
		egressPolicies = egressPolicyList.EgressPolicies
		Expect(egressPolicies).To(HaveLen(3))
		Expect(egressPolicies).To(WithTransform(replaceEgressPoliciesGUID, ConsistOf(expectedSomePolicy, expectedSomeThirdPolicy, expectedPolicyWithAllProtocols)))

		By("ANDing search filter params")
		egressPolicyList, err = client.ListEgressPolicies(token, []string{"live-app-1-guid", "live-app-3-guid"}, []string{"app"}, []string{createdDestinations[0].GUID}, []string{})
		Expect(err).NotTo(HaveOccurred())
		egressPolicies = egressPolicyList.EgressPolicies
		Expect(egressPolicies).To(HaveLen(1))
		Expect(egressPolicies).To(WithTransform(replaceEgressPoliciesGUID, ConsistOf(expectedSomePolicy)))

		By("deleting destinations and policies")
		_, err = client.DeleteDestination(token, createdDestinations[0])
		Expect(err).To(HaveOccurred(), "expected the delete to fail because this destinations still has associated egress policy")
		Expect(err).To(MatchError(ContainSubstring("destination is still in use")))

		_, err = client.DeleteEgressPolicy(policyGUID, token)
		Expect(err).NotTo(HaveOccurred())

		_, err = client.DeleteEgressPolicy(secondPolicyGUID, token)
		Expect(err).NotTo(HaveOccurred())

		_, err = client.DeleteEgressPolicy(thirdPolicyGUID, token)
		Expect(err).NotTo(HaveOccurred())

		_, err = client.DeleteEgressPolicy(policyWithAllProtocolsGUID, token)
		Expect(err).NotTo(HaveOccurred())

		egressPolicyList, err = client.ListEgressPolicies(token, []string{}, []string{}, []string{}, []string{})
		Expect(err).NotTo(HaveOccurred())
		egressPolicies = egressPolicyList.EgressPolicies
		Expect(egressPolicies).To(HaveLen(0))
	})
})
