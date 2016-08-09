package planner_test

import (
	"encoding/json"
	"lib/rules"
	"natman/planner"
	"net"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden/gardenfakes"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

func netOutRuleToString(rules []garden.NetOutRule) string {
	rulesBytes, err := json.Marshal(rules)
	Expect(err).NotTo(HaveOccurred())
	return string(rulesBytes)
}

var _ = Describe("NetOutPlanner", func() {
	var (
		p              *planner.NetOutPlanner
		fakeClient     *gardenfakes.FakeClient
		fakeContainer1 *gardenfakes.FakeContainer
		fakeContainer2 *gardenfakes.FakeContainer
		logger         *lagertest.TestLogger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		fakeContainer1 = &gardenfakes.FakeContainer{}
		fakeContainer2 = &gardenfakes.FakeContainer{}
		fakeClient = &gardenfakes.FakeClient{}
		p = &planner.NetOutPlanner{
			GardenClient:   fakeClient,
			OverlayNetwork: "10.255.0.0/16",
			Logger:         logger,
		}
	})

	Describe("GetRules", func() {
		BeforeEach(func() {
			rules := []garden.NetOutRule{{
				Networks: []garden.IPRange{{
					Start: net.ParseIP("1.2.3.4"),
					End:   net.ParseIP("5.6.7.8"),
				}},
			}, {
				Networks: []garden.IPRange{{
					Start: net.ParseIP("1.1.1.1"),
					End:   net.ParseIP("2.2.2.2"),
				}},
				Ports: []garden.PortRange{
					{Start: 123, End: 9999},
					{Start: 53, End: 53},
				},
				Protocol: garden.ProtocolTCP,
			},
			}
			fakeContainer1.InfoReturns(garden.ContainerInfo{
				ContainerIP: "10.255.1.2",
				ExternalIP:  "10.254.16.2",
				Properties: map[string]string{
					"network.app_id":                     "some-app-guid",
					"network.external-networker.net-out": netOutRuleToString(rules),
				},
			}, nil)

			fakeClient.ContainersReturns([]garden.Container{
				fakeContainer1,
			}, nil)
		})

		It("returns a list of rules with a default deny appended", func() {
			r, err := p.GetRules()
			Expect(err).NotTo(HaveOccurred())
			Expect(r).To(ConsistOf([]rules.GenericRule{{
				Properties: []string{
					"--source", "10.255.1.2",
					"-m", "iprange",
					"--dst-range", "1.2.3.4-5.6.7.8",
					"--jump", "RETURN",
					"-m", "comment", "--comment", "dst:some-app-guid",
				},
			}, {
				Properties: []string{
					"--source", "10.255.1.2",
					"-m", "iprange",
					"-p", "tcp",
					"--dst-range", "1.1.1.1-2.2.2.2",
					"-m", "tcp",
					"--destination-port", "123:9999",
					"--jump", "RETURN",
					"-m", "comment", "--comment", "dst:some-app-guid",
				},
			}, {
				Properties: []string{
					"--source", "10.255.1.2",
					"-m", "iprange",
					"-p", "tcp",
					"--dst-range", "1.1.1.1-2.2.2.2",
					"-m", "tcp",
					"--destination-port", "53:53",
					"--jump", "RETURN",
					"-m", "comment", "--comment", "dst:some-app-guid",
				},
			}, {
				Properties: []string{
					"-s", "10.255.0.0/16",
					"!", "-d", "10.255.0.0/16",
					"-m", "state", "--state", "RELATED,ESTABLISHED",
					"--jump", "RETURN",
				},
			}, {
				Properties: []string{
					"-s", "10.255.0.0/16",
					"!", "-d", "10.255.0.0/16",
					"--jump", "REJECT",
					"--reject-with", "icmp-port-unreachable",
				},
			},
			}))
		})

		Context("when thenetout rules key is empty", func() {
			BeforeEach(func() {
				fakeContainer1.InfoReturns(garden.ContainerInfo{
					ContainerIP: "10.255.1.2",
					ExternalIP:  "10.254.16.2",
					Properties: map[string]string{
						"network.app_id":                     "some-app-guid",
						"network.external-networker.net-out": "",
					},
				}, nil)

				fakeClient.ContainersReturns([]garden.Container{
					fakeContainer1,
				}, nil)
			})
			It("skips that container", func() {
				r, err := p.GetRules()
				Expect(err).NotTo(HaveOccurred())
				Expect(r).To(ConsistOf([]rules.GenericRule{
					{
						Properties: []string{
							"-s", "10.255.0.0/16",
							"!", "-d", "10.255.0.0/16",
							"-m", "state", "--state", "RELATED,ESTABLISHED",
							"--jump", "RETURN",
						},
					}, {
						Properties: []string{
							"-s", "10.255.0.0/16",
							"!", "-d", "10.255.0.0/16",
							"--jump", "REJECT",
							"--reject-with", "icmp-port-unreachable",
						},
					},
				}))
			})
		})

		Context("when there's no netout rules key", func() {
			BeforeEach(func() {
				fakeContainer1.InfoReturns(garden.ContainerInfo{
					ContainerIP: "10.255.1.2",
					ExternalIP:  "10.254.16.2",
					Properties: map[string]string{
						"network.app_id": "some-app-guid",
					},
				}, nil)

				fakeClient.ContainersReturns([]garden.Container{
					fakeContainer1,
				}, nil)
			})
			It("skips that container", func() {
				r, err := p.GetRules()
				Expect(err).NotTo(HaveOccurred())
				Expect(r).To(ConsistOf([]rules.GenericRule{
					{
						Properties: []string{
							"-s", "10.255.0.0/16",
							"!", "-d", "10.255.0.0/16",
							"-m", "state", "--state", "RELATED,ESTABLISHED",
							"--jump", "RETURN",
						},
					}, {
						Properties: []string{
							"-s", "10.255.0.0/16",
							"!", "-d", "10.255.0.0/16",
							"--jump", "REJECT",
							"--reject-with", "icmp-port-unreachable",
						},
					},
				}))
			})
		})

		Context("when unmarshaling the netout properties fails", func() {
			BeforeEach(func() {
				fakeContainer1.InfoReturns(garden.ContainerInfo{
					ContainerIP: "10.255.1.2",
					ExternalIP:  "10.254.16.2",
					Properties: map[string]string{
						"network.app_id":                     "some-app-guid",
						"network.external-networker.net-out": "%%%%%",
					},
				}, nil)

				fakeClient.ContainersReturns([]garden.Container{
					fakeContainer1,
				}, nil)
			})
			It("logs and returns an error", func() {
				_, err := p.GetRules()

				Expect(logger).To(gbytes.Say("netout-unmarshal-json.*invalid character"))
				Expect(err).To(MatchError(ContainSubstring("invalid character")))
			})
		})
	})
})
