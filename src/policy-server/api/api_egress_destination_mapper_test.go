package api_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"policy-server/api"
	"policy-server/store"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
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
					GUID:     "1",
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
					GUID:        "2",
					Description: " ",
					Protocol:    "icmp",
					IPRanges: []store.IPRange{{
						Start: "1.2.3.7",
						End:   "1.2.3.8",
					}},
					ICMPType: 1,
					ICMPCode: 6,
				},
				{
					GUID:     "3",
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
							"id": "1",
							"name": " ",
							"protocol": "tcp",
							"ports": [{ "start": 8080, "end": 8081 }],
							"ips": [{ "start": "1.2.3.4", "end": "1.2.3.5" }]
						},
						{
							"id": "2",
							"description": " ",
							"protocol": "icmp",
							"ips": [{ "start": "1.2.3.7", "end": "1.2.3.8" }],
							"icmp_type": 1,
							"icmp_code": 6
						},
 						{
							"id": "3",
							"protocol": "udp",
							"ips": [{ "start": "1.2.3.7", "end": "1.2.3.8" }]
						}
					]
				}`)))
		})
	})

	Describe("AsEgressDestinations", func() {
		var expectedOutputBytes []byte

		BeforeEach(func() {
			expectedOutputBytes = []byte(`{
					"total_destinations": 3,
					"destinations": [
						{
							"id": "1",
							"name": "my service",
							"protocol": "tcp",
							"ports": [{ "start": 8080, "end": 8081 }],
							"ips": [{ "start": "1.2.3.4", "end": "1.2.3.5" }]
						},
						{
							"id": "2",
							"description": "this is where my apps go",
							"protocol": "icmp",
							"ips": [{ "start": "1.2.3.7", "end": "1.2.3.8" }],
							"icmp_type": 1,
							"icmp_code": 6
						},
 						{
							"id": "3",
							"protocol": "udp",
							"ips": [{ "start": "1.2.3.7", "end": "1.2.3.8" }]
						}
					]
				}`)
		})

		It("unmarshals egress destinations from json", func() {
			payload, err := mapper.AsEgressDestinations(expectedOutputBytes)
			Expect(err).NotTo(HaveOccurred())
			Expect(payload).To(Equal(
				[]store.EgressDestination{
					{
						GUID:     "1",
						Name:     "my service",
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
						GUID:        "2",
						Description: "this is where my apps go",
						Protocol:    "icmp",
						Ports:       []store.Ports{},
						IPRanges: []store.IPRange{{
							Start: "1.2.3.7",
							End:   "1.2.3.8",
						}},
						ICMPType: 1,
						ICMPCode: 6,
					},
					{
						GUID:     "3",
						Protocol: "udp",
						Ports:    []store.Ports{},
						IPRanges: []store.IPRange{{
							Start: "1.2.3.7",
							End:   "1.2.3.8",
						}},
					},
				}),
			)
		})
	})
})
