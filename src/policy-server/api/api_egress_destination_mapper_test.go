package api_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"encoding/json"
	"policy-server/api"
	"policy-server/store"
)

var _ = Describe("ApiEgressDestinationMapper", func() {
	var (
		mapper *api.EgressDestinationMapper
	)

	BeforeEach(func() {
		mapper = &api.EgressDestinationMapper{
			Marshaler: marshal.MarshalFunc(json.Marshal),
		}
	})

	Describe("AsBytes", func() {
		var egressDestinations []store.EgressDestination
		BeforeEach(func() {
			egressDestinations = []store.EgressDestination{
				{
					ID:       "1",
					Name:     " ",
					Protocol: "tcp",
					Ports: []store.Ports{{
						Start: 8080,
						End:   8081,
					}},
					IPRanges: []store.IPRange{{
						Start: "1.2.3.4",
						End:   "1.2.3.5",
					}},
				},
				{
					ID:       "2",
					Description: " ",
					Protocol: "icmp",
					IPRanges: []store.IPRange{{
						Start: "1.2.3.7",
						End:   "1.2.3.8",
					}},
					ICMPType: 1,
					ICMPCode: 6,
				},
				{
					ID:       "3",
					Protocol: "udp",
					IPRanges: []store.IPRange{{
						Start: "1.2.3.7",
						End:   "1.2.3.8",
					}},
				},
			}
		})

		It("marshals to json", func() {
			payload, err := mapper.AsBytes(egressDestinations)
			Expect(err).NotTo(HaveOccurred())
			Expect(payload).To(MatchJSON(
				[]byte(`{
					"total_destinations": 3,
					"destinations": [
						{   
							"guid": "1",
							"name": " ",
							"protocol": "tcp",
							"ports": [{ "start": 8080, "end": 8081 }],
							"ips": [{ "start": "1.2.3.4", "end": "1.2.3.5" }]
						},
						{
							"guid": "2",
							"description": " ",
							"protocol": "icmp",
							"ips": [{ "start": "1.2.3.7", "end": "1.2.3.8" }],
							"icmp_type": 1,
							"icmp_code": 6
						},
 						{
							"guid": "3",
							"protocol": "udp",
							"ips": [{ "start": "1.2.3.7", "end": "1.2.3.8" }]
						}
					]
				}`)))
		})
	})
})
