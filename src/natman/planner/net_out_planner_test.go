package planner_test

import (
	"encoding/json"
	"natman/planner"
	"net"
	"netman-agent/rules"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden/gardenfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func netOutRuleToString(rules []garden.NetOutRule) string {
	rulesBytes, err := json.Marshal(rules)
	Expect(err).NotTo(HaveOccurred())
	return string(rulesBytes)
}

var _ = Describe("NetOutPlanner", func() {
	var p *planner.NetOutPlanner
	var fakeClient *gardenfakes.FakeClient
	var fakeContainer1 *gardenfakes.FakeContainer
	var fakeContainer2 *gardenfakes.FakeContainer

	BeforeEach(func() {
		fakeContainer1 = &gardenfakes.FakeContainer{}
		fakeContainer2 = &gardenfakes.FakeContainer{}
		fakeClient = &gardenfakes.FakeClient{}
		p = &planner.NetOutPlanner{
			GardenClient:   fakeClient,
			OverlayNetwork: "10.255.0.0/16",
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
					"!", "-d", "10.255.0.0/16",
					"--jump", "REJECT",
					"--reject-with", "icmp-port-unreachable",
				},
			},
			}))
		})
	})
})
