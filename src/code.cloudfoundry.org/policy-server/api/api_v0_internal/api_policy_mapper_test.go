package api_v0_internal_test

import (
	"encoding/json"
	"errors"

	"code.cloudfoundry.org/cf-networking-helpers/fakes"
	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/policy-server/api"
	"code.cloudfoundry.org/policy-server/api/api_v0_internal"
	"code.cloudfoundry.org/policy-server/store"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ApiPolicyMapper v000", func() {
	var (
		mapper        api.PolicyMapper
		fakeMarshaler *fakes.Marshaler
	)
	BeforeEach(func() {
		mapper = api_v0_internal.NewMapper(
			marshal.UnmarshalFunc(json.Unmarshal),
			marshal.MarshalFunc(json.Marshal),
		)
		fakeMarshaler = &fakes.Marshaler{}
	})
	Describe("AsBytes", func() {
		It("maps a slice of store.Policy to a payload with api.Policy", func() {
			payload, err := mapper.AsBytes([]store.Policy{
				{
					Source: store.Source{ID: "some-src-id"},
					Destination: store.Destination{
						ID:       "some-dst-id",
						Tag:      "some-other-dst-tag",
						Protocol: "some-protocol",
						Port:     8080,
						Ports: store.Ports{
							Start: 8080,
							End:   8080,
						},
					},
				}, {
					Source: store.Source{ID: "some-src-id-2"},
					Destination: store.Destination{
						ID:       "some-dst-id-2",
						Tag:      "some-other-dst-tag-2",
						Protocol: "some-protocol-2",
						Port:     8081,
						Ports: store.Ports{
							Start: 8081,
							End:   8081,
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(payload).To(MatchJSON(
				[]byte(`{
					"total_policies": 2,
					"policies": [{
						"source": { "id": "some-src-id" },
						"destination": {
							"id": "some-dst-id",
							"tag": "some-other-dst-tag",
							"protocol": "some-protocol",
							"port": 8080,
							"ports": { "start": 8080, "end": 8080 }
						}
					}, {
						"source": { "id": "some-src-id-2" },
						"destination": {
							"id": "some-dst-id-2",
							"tag": "some-other-dst-tag-2",
							"protocol": "some-protocol-2",
							"port": 8081,
							"ports": { "start": 8081, "end": 8081 }
						}
					}]
				}`),
			))
		})
		Context("when the policy has no port but start and end port are the same", func() {
			It("uses the value from the ports field", func() {
				payload, err := mapper.AsBytes([]store.Policy{
					{
						Source: store.Source{ID: "some-src-id"},
						Destination: store.Destination{
							ID:       "some-dst-id",
							Tag:      "some-other-dst-tag",
							Protocol: "some-protocol",
							Ports: store.Ports{
								Start: 8080,
								End:   8080,
							},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(payload).To(MatchJSON([]byte(`{
					"total_policies": 1,
					"policies": [
						{
							"source": { "id": "some-src-id" },
							"destination": {
								"id": "some-dst-id",
								"tag": "some-other-dst-tag",
								"protocol": "some-protocol",
								"port": 8080,
							  "ports": { "start": 8080, "end": 8080 }
							}
						}
					]
				}`)))
			})
		})
		Context("when the policy has an empty tag", func() {
			It("omits the tag field", func() {
				payload, err := mapper.AsBytes([]store.Policy{
					{
						Source: store.Source{ID: "some-src-id"},
						Destination: store.Destination{
							ID:       "some-dst-id",
							Protocol: "some-protocol",
							Ports: store.Ports{
								Start: 8080,
								End:   8080,
							},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(payload).To(MatchJSON([]byte(`{
					"total_policies": 1,
					"policies": [
						{
							"source": { "id": "some-src-id" },
							"destination": {
								"id": "some-dst-id",
								"protocol": "some-protocol",
								"port": 8080,
							  "ports": { "start": 8080, "end": 8080 }
							}
						}
					]
				}`)))
			})
		})
		Context("when the destination.StartPort does not equal destination.EndPort", func() {
			It("ignores a store.Policy that cannot be mapped to an api.Policy", func() {
				payload, err := mapper.AsBytes([]store.Policy{
					{
						Source: store.Source{ID: "some-src-id"},
						Destination: store.Destination{
							ID:       "some-dst-id",
							Tag:      "some-other-dst-tag",
							Protocol: "some-protocol",
							Ports: store.Ports{
								Start: 8070,
								End:   8080,
							},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(payload).To(MatchJSON([]byte(`{ "total_policies": 0, "policies": [] }`)))
			})
		})
		Context("when marshalling fails", func() {
			BeforeEach(func() {
				fakeMarshaler.MarshalReturns(nil, errors.New("banana"))
				mapper = api_v0_internal.NewMapper(
					marshal.UnmarshalFunc(json.Unmarshal),
					fakeMarshaler,
				)
			})
			It("wraps and returns an error", func() {
				_, err := mapper.AsBytes([]store.Policy{})
				Expect(err).To(MatchError(errors.New("marshal json: banana")))
			})
		})
	})
})
