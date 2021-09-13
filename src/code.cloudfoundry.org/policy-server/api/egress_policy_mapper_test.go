package api_test

import (
	"encoding/json"
	"errors"

	hfakes "code.cloudfoundry.org/cf-networking-helpers/fakes"
	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/policy-server/api"
	"code.cloudfoundry.org/policy-server/api/fakes"
	"code.cloudfoundry.org/policy-server/store"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EgressPolicyMapper", func() {
	var (
		mapper    *api.EgressPolicyMapper
		validator *fakes.EgressValidator
	)

	BeforeEach(func() {
		validator = &fakes.EgressValidator{}

		mapper = &api.EgressPolicyMapper{
			Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
			Marshaler:   marshal.MarshalFunc(json.Marshal),
			Validator:   validator,
		}
	})

	Describe("AsStoreEgressPolicy", func() {
		It("maps a payload with api.EgressPolicy to a slice of store.EgressPolicy", func() {
			payloadBytes := []byte(`{
				"egress_policies": [
                    {
						"source": { "id": "some-src-id", "type": "app" },
						"destination": { "id": "some-dst-id" },
						"app_lifecycle": "all"
					},
                    {
						"source": { "id": "some-src-id-2", "type": "space"  },
						"destination": { "id": "some-dst-id-2" }
					}
				]
			}`)

			policies, err := mapper.AsStoreEgressPolicy(payloadBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(policies).To(HaveLen(2))
			Expect(policies[0].Source.ID).To(Equal("some-src-id"))
			Expect(policies[0].Source.Type).To(Equal("app"))
			Expect(policies[0].Destination.GUID).To(Equal("some-dst-id"))
			Expect(policies[0].AppLifecycle).To(Equal("all"))
			Expect(policies[1].Source.ID).To(Equal("some-src-id-2"))
			Expect(policies[1].Source.Type).To(Equal("space"))
			Expect(policies[1].Destination.GUID).To(Equal("some-dst-id-2"))
			Expect(policies[1].AppLifecycle).To(Equal("all"))
		})

		Context("when unmarshalling fails", func() {
			It("wraps and returns an error", func() {
				_, err := mapper.AsStoreEgressPolicy([]byte("garbage"))
				Expect(err).To(MatchError(errors.New("unmarshal json: invalid character 'g' looking for beginning of value")))
			})
		})

		Context("when validation fails", func() {
			BeforeEach(func() {
				validator.ValidateEgressPoliciesReturns(errors.New("does not validate"))
			})
			It("wraps and returns an error", func() {
				payloadBytes := []byte(`{
					"egress_policies": [
						{
							"source": { "id": "some-src-id", "type": "app" },
							"destination": { "id": "some-dst-id" }
						}
					]
				}`)
				_, err := mapper.AsStoreEgressPolicy(payloadBytes)

				Expect(err).To(MatchError(errors.New("validating egress policies: does not validate")))
			})
		})
	})

	Describe("AsBytes", func() {
		var egressPolicies []store.EgressPolicy

		BeforeEach(func() {
			egressPolicies = []store.EgressPolicy{
				{
					ID:     "policy-1",
					Source: store.EgressSource{ID: "some-src-id", Type: "app"},
					Destination: store.EgressDestination{
						GUID: "some-dst-id",
					},
					AppLifecycle: "running",
				},
				{
					ID:     "policy-2",
					Source: store.EgressSource{ID: "some-src-id-2", Type: "space"},
					Destination: store.EgressDestination{
						GUID:        "some-dst-id-2",
						Name:        "dest-name",
						Description: "dest-desc",
						Rules: []store.EgressDestinationRule{
							{
								IPRanges: []store.IPRange{{Start: "1.1.1.1", End: "2.2.2.2"}},
								Ports:    []store.Ports{{Start: 1212, End: 2323}},
								Protocol: "icmp",
								ICMPType: 4,
								ICMPCode: 3,
							},
						},
					},
					AppLifecycle: "staging",
				},
			}
		})

		It("maps a payload with api.EgressPolicy to a slice of store.EgressPolicy", func() {
			mappedBytes, err := mapper.AsBytes(egressPolicies)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(mappedBytes)).To(MatchJSON(`{
					"total_egress_policies": 2,
					"egress_policies": [
            	        {
							"id": "policy-1",
							"source": { "id": "some-src-id", "type": "app" },
							"destination": { "id": "some-dst-id" },
							"app_lifecycle": "running"
						},
               	    	{
							"id": "policy-2",
							"source": { "id": "some-src-id-2", "type": "space" },
							"destination": { "id": "some-dst-id-2" },
							"app_lifecycle": "staging"
						}
					]
				}`))
		})
		Context("when marshalling fails", func() {
			BeforeEach(func() {
				marshaler := &hfakes.Marshaler{}
				marshaler.MarshalReturns([]byte{}, errors.New("failed to marshal bytes"))
				mapper.Marshaler = marshaler
			})

			It("wraps and returns an error", func() {
				_, err := mapper.AsBytes(egressPolicies)
				Expect(err).To(MatchError(errors.New("marshal json: failed to marshal bytes")))
			})
		})
	})

	Describe("AsBytesWithPopulatedDestinations", func() {
		var egressPolicies []store.EgressPolicy

		BeforeEach(func() {
			egressPolicies = []store.EgressPolicy{
				{
					ID:     "policy-1",
					Source: store.EgressSource{ID: "some-src-id", Type: "app"},
					Destination: store.EgressDestination{
						GUID: "some-dst-id",
						Rules: []store.EgressDestinationRule{
							{
								IPRanges: []store.IPRange{{Start: "2.1.1.1", End: "3.2.2.2"}},
							},
						},
					},
					AppLifecycle: "running",
				},
				{
					ID:     "policy-2",
					Source: store.EgressSource{ID: "some-src-id-2", Type: "space"},
					Destination: store.EgressDestination{
						GUID:        "some-dst-id-2",
						Name:        "dest-name",
						Description: "dest-desc",
						Rules: []store.EgressDestinationRule{
							{
								IPRanges: []store.IPRange{{Start: "1.1.1.1", End: "2.2.2.2"}},
								Ports:    []store.Ports{{Start: 1212, End: 2323}},
								Protocol: "icmp",
								ICMPType: 4,
								ICMPCode: 3,
							},
						},
					},
					AppLifecycle: "staging",
				},
			}
		})

		It("maps a payload with api.EgressPolicy to a slice of store.EgressPolicy", func() {
			mappedBytes, err := mapper.AsBytesWithPopulatedDestinations(egressPolicies)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(mappedBytes)).To(MatchJSON(`{
					"total_egress_policies": 2,
					"egress_policies": [
            	        {
							"id": "policy-1",
							"source": { "id": "some-src-id", "type": "app" },
							"destination": {
								"id": "some-dst-id",
								"rules": [ { "ips": "2.1.1.1-3.2.2.2", "description": "" } ]
							},
							"app_lifecycle": "running"
						},
               	    	{
							"id": "policy-2",
							"source": { "id": "some-src-id-2", "type": "space" },
							"destination": {
								"id": "some-dst-id-2",
								"name": "dest-name",
								"description": "dest-desc",
								"rules": [
									{
										"description": "",
										"ips": "1.1.1.1-2.2.2.2",
										"ports": "1212-2323",
										"protocol": "icmp",
										"icmp_type": 4,
										"icmp_code": 3
									}
								]
							},
							"app_lifecycle": "staging"
						}
					]
				}`))
		})

		Context("when marshalling fails", func() {
			BeforeEach(func() {
				marshaler := &hfakes.Marshaler{}
				marshaler.MarshalReturns([]byte{}, errors.New("failed to marshal bytes"))
				mapper.Marshaler = marshaler
			})

			It("wraps and returns an error", func() {
				_, err := mapper.AsBytesWithPopulatedDestinations(egressPolicies)
				Expect(err).To(MatchError(errors.New("marshal json: failed to marshal bytes")))
			})
		})

		Context("when there are no egress policies", func() {
			BeforeEach(func() {
				egressPolicies = []store.EgressPolicy{}
			})

			It("returns keys with no values", func() {
				mappedBytes, err := mapper.AsBytesWithPopulatedDestinations(egressPolicies)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(mappedBytes)).To(MatchJSON(`{
					"total_egress_policies": 0,
					"egress_policies": []
				}`))
			})
		})
	})
})
