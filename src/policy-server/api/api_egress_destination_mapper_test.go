package api_test

import (
	"encoding/json"
	"errors"
	"policy-server/api"
	"policy-server/api/fakes"
	"policy-server/store"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ApiEgressDestinationMapper", func() {
	var (
		mapper        *api.EgressDestinationMapper
		fakeValidator *fakes.EgressDestinationsValidator
	)

	BeforeEach(func() {
		fakeValidator = &fakes.EgressDestinationsValidator{}
		fakeValidator.ValidateEgressDestinationsReturns(nil)

		mapper = &api.EgressDestinationMapper{
			Marshaler:        marshal.MarshalFunc(json.Marshal),
			PayloadValidator: fakeValidator,
		}
	})

	Describe("AsBytes", func() {
		var egressDestinations []store.EgressDestination
		BeforeEach(func() {
			egressDestinations = []store.EgressDestination{
				{
					GUID: "1",
					Name: " ",
					Rules: []store.EgressDestinationRule{
						{
							Description: "hello friends",
							Protocol:    "tcp",
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
							Protocol: "udp",
							IPRanges: []store.IPRange{{
								Start: "10.20.30.40",
								End:   "10.20.30.50",
							}},
						},
						{
							Description: "marshal me",
							Protocol:    "icmp",
							IPRanges: []store.IPRange{{
								Start: "10.20.30.40",
								End:   "10.20.30.50",
							}},
							ICMPType: 2,
							ICMPCode: 3,
						},
					},
				},
				{
					GUID:        "2",
					Description: " ",
					Rules: []store.EgressDestinationRule{
						{
							Protocol: "icmp",
							IPRanges: []store.IPRange{{
								Start: "1.2.3.7",
								End:   "1.2.3.8",
							}},
							ICMPType: 1,
							ICMPCode: 6,
						},
					},
				},
				{
					GUID: "3",
					Rules: []store.EgressDestinationRule{
						{
							Protocol: "udp",
							IPRanges: []store.IPRange{{
								Start: "1.2.3.7",
								End:   "1.2.3.8",
							}},
						},
					},
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
							"rules": [
								{
									"description": "hello friends",
									"protocol": "tcp",
									"ports": [{ "start": 8080, "end": 8081 }],
									"ips": "1.2.3.4-1.2.3.5"
								},
								{
									"description": "",
									"protocol": "udp",
									"ips": "10.20.30.40-10.20.30.50"
								},
								{
									"description": "marshal me",
									"protocol": "icmp",
									"ips": "10.20.30.40-10.20.30.50",
									"icmp_type": 2,
									"icmp_code": 3
								}
							]
						},
						{
							"id": "2",
							"description": " ",
							"rules": [
								{
									"description": "",
									"protocol": "icmp",
									"ips": "1.2.3.7-1.2.3.8",
									"icmp_type": 1,
									"icmp_code": 6
								}
							]
						},
 						{
							"id": "3",
							"rules": [
								{
									"description": "",
									"protocol": "udp",
									"ips": "1.2.3.7-1.2.3.8"
								}
							]
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
							"rules": [
								{
									"description": "hamdinger",
									"protocol": "tcp",
									"ports": [{ "start": 8080, "end": 8081 }],
									"ips": "1.2.3.4-1.2.3.5"
								},
								{
									"protocol": "icmp",
									"ips": "10.20.30.70-10.20.30.80",
									"icmp_type": 2,
									"icmp_code": 3
								}
							]
						},
						{
							"id": "2",
							"description": "this is where my apps go",
							"rules": [
								{
									"description": "rule 2 by blur",
									"protocol": "icmp",
									"ips": "1.2.3.7-1.2.3.8",
									"icmp_type": 1,
									"icmp_code": 6
								}
							]
						},
						{
							"id": "3",
							"description": "regression test: icmp without type and code",
							"rules": [
								{
									"protocol": "icmp",
									"ips": "1.2.3.7-1.2.3.8"
								}
							]
						},
						{
							"id": "4",
							"rules": [
								{
									"protocol": "udp",
									"ips": "1.2.3.7-1.2.3.8"
								}
							]
						},
						{
							"id": "5",
							"rules": [
								{
									"description": "the fifth rule sorry not sorry",
									"protocol": "all",
									"ips": "1.2.3.7-1.2.3.8"
								}
							]
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
						GUID: "1",
						Name: "my service",
						Rules: []store.EgressDestinationRule{
							{
								Description: "hamdinger",
								Protocol:    "tcp",
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
								Protocol: "icmp",
								Ports:    []store.Ports{},
								IPRanges: []store.IPRange{{
									Start: "10.20.30.70",
									End:   "10.20.30.80",
								}},
								ICMPType: 2,
								ICMPCode: 3,
							},
						},
					},
					{
						GUID:        "2",
						Description: "this is where my apps go",
						Rules: []store.EgressDestinationRule{
							{
								Description: "rule 2 by blur",
								Protocol:    "icmp",
								Ports:       []store.Ports{},
								IPRanges: []store.IPRange{{
									Start: "1.2.3.7",
									End:   "1.2.3.8",
								}},
								ICMPType: 1,
								ICMPCode: 6,
							},
						},
					},
					{
						GUID:        "3",
						Description: "regression test: icmp without type and code",
						Rules: []store.EgressDestinationRule{
							{
								Protocol: "icmp",
								Ports:    []store.Ports{},
								IPRanges: []store.IPRange{{
									Start: "1.2.3.7",
									End:   "1.2.3.8",
								}},
								ICMPType: -1,
								ICMPCode: -1,
							},
						},
					},
					{
						GUID: "4",
						Rules: []store.EgressDestinationRule{
							{
								Protocol: "udp",
								Ports:    []store.Ports{},
								IPRanges: []store.IPRange{{
									Start: "1.2.3.7",
									End:   "1.2.3.8",
								}},
							},
						},
					},
					{
						GUID: "5",
						Rules: []store.EgressDestinationRule{
							{
								Description: "the fifth rule sorry not sorry",
								Protocol:    "all",
								Ports:       []store.Ports{},
								IPRanges: []store.IPRange{{
									Start: "1.2.3.7",
									End:   "1.2.3.8",
								}},
							},
						},
					},
				}),
			)
		})

		Context("when there is a json unmarshalling error", func() {
			It("returns an error", func() {
				_, err := mapper.AsEgressDestinations([]byte("%%%"))
				Expect(err).To(MatchError("unmarshal json: invalid character '%' looking for beginning of value"))
			})
		})

		Context("when there is a validation error", func() {
			BeforeEach(func() {
				fakeValidator.ValidateEgressDestinationsReturns(errors.New("banana"))
			})

			It("returns an error", func() {
				_, err := mapper.AsEgressDestinations(expectedOutputBytes)
				Expect(err).To(MatchError("validate destinations: banana"))
			})
		})
	})
})
