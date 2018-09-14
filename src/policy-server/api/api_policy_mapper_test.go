package api_test

import (
	"encoding/json"
	"errors"
	"policy-server/api"
	"policy-server/store"

	"policy-server/api/fakes"

	hfakes "code.cloudfoundry.org/cf-networking-helpers/fakes"
	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("ApiPolicyMapper", func() {
	var (
		mapper          api.PolicyMapper
		fakeUnmarshaler *hfakes.Unmarshaler
		fakeMarshaler   *hfakes.Marshaler
		fakeValidator   *fakes.PayloadValidator
	)
	BeforeEach(func() {
		fakeValidator = &fakes.PayloadValidator{}
		mapper = api.NewMapper(
			marshal.UnmarshalFunc(json.Unmarshal),
			marshal.MarshalFunc(json.Marshal),
			fakeValidator,
		)
		fakeUnmarshaler = &hfakes.Unmarshaler{}
		fakeMarshaler = &hfakes.Marshaler{}
	})
	Describe("AsStorePolicy", func() {
		It("maps a payload with api.Policy to a slice of store.Policy", func() {
			policies, err := mapper.AsStorePolicy(
				[]byte(`{
					"policies": [{
						"source": { "id": "some-src-id" },
						"destination": {
							"id": "some-dst-id",
							"tag": "some-other-dst-tag",
							"protocol": "some-protocol",
							"ports": {
								"start": 8080,
								"end": 9090
							}
						}
					}, {
						"source": { "id": "some-src-id-2" },
						"destination": {
							"id": "some-dst-id-2",
							"tag": "some-other-dst-tag-2",
							"protocol": "some-protocol-2",
							"ports": {
								"start": 8080,
								"end": 8080
							}
						}
					}]
				}`),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeValidator.ValidatePayloadCallCount()).To(Equal(1))
			Expect(fakeValidator.ValidatePayloadArgsForCall(0)).To(Equal(
				&api.PoliciesPayload{
					Policies: []api.Policy{
						{
							Source: api.Source{ID: "some-src-id", Tag: ""},
							Destination: api.Destination{
								ID:       "some-dst-id",
								Tag:      "some-other-dst-tag",
								Protocol: "some-protocol",
								Ports:    api.Ports{Start: 8080, End: 9090},
							},
						},
						{
							Source: api.Source{ID: "some-src-id-2", Tag: ""},
							Destination: api.Destination{
								ID:       "some-dst-id-2",
								Tag:      "some-other-dst-tag-2",
								Protocol: "some-protocol-2",
								Ports:    api.Ports{Start: 8080, End: 8080},
							},
						},
					},
				},
			))
			Expect(policies).To(Equal([]store.Policy{
				{
					Source: store.Source{ID: "some-src-id"},
					Destination: store.Destination{
						ID:       "some-dst-id",
						Tag:      "some-other-dst-tag",
						Protocol: "some-protocol",
						Port:     0,
						Ports: store.Ports{
							Start: 8080,
							End:   9090,
						},
					},
				}, {
					Source: store.Source{ID: "some-src-id-2"},
					Destination: store.Destination{
						ID:       "some-dst-id-2",
						Tag:      "some-other-dst-tag-2",
						Protocol: "some-protocol-2",
						Port:     8080,
						Ports: store.Ports{
							Start: 8080,
							End:   8080,
						},
					},
				},
			}))
		})



		Context("when unmarshalling fails", func() {
			BeforeEach(func() {
				fakeUnmarshaler.UnmarshalReturns(errors.New("banana"))
				mapper = api.NewMapper(
					fakeUnmarshaler,
					marshal.MarshalFunc(json.Marshal),
					fakeValidator,
				)
			})
			It("wraps and returns an error", func() {
				_, err := mapper.AsStorePolicy([]byte("somebytes"))
				Expect(err).To(MatchError(errors.New("unmarshal json: banana")))
			})
		})

		Context("when a policy to includes a validation error", func() {
			BeforeEach(func() {
				fakeValidator.ValidatePayloadReturns(errors.New("banana"))
			})

			It("calls the bad request handler", func() {
				_, err := mapper.AsStorePolicy([]byte(`{}`))
				Expect(err).To(MatchError(errors.New("validate policies: banana")))
			})
		})
	})

	Describe("AsBytes", func() {
		It("maps a slice of store.Policy to a payload with api.Policy", func() {
			policies := []store.Policy{
				{
					Source: store.Source{ID: "some-src-id"},
					Destination: store.Destination{
						ID:       "some-dst-id",
						Tag:      "some-other-dst-tag",
						Protocol: "some-protocol",
						Ports: store.Ports{
							Start: 8080,
							End:   9090,
						},
					},
				}, {
					Source: store.Source{ID: "some-src-id-2"},
					Destination: store.Destination{
						ID:       "some-dst-id-2",
						Tag:      "some-other-dst-tag-2",
						Protocol: "some-protocol-2",
						Ports: store.Ports{
							Start: 8081,
							End:   8081,
						},
					},
				},
			}


			payload, err := mapper.AsBytes(policies)
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
							"ports": {
								"start": 8080,
								"end": 9090
							}
						}
					}, {
						"source": { "id": "some-src-id-2" },
						"destination": {
							"id": "some-dst-id-2",
							"tag": "some-other-dst-tag-2",
							"protocol": "some-protocol-2",
							"ports": {
								"start": 8081,
								"end": 8081
							}
						}
					}]
				}`),
			))
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
								"ports": {
									"start": 8080,
									"end": 8080
								}
							}
						}
					]
				}`)))
			})
		})
		Context("when marshalling fails", func() {
			BeforeEach(func() {
				fakeMarshaler.MarshalReturns(nil, errors.New("banana"))
				mapper = api.NewMapper(
					marshal.UnmarshalFunc(json.Unmarshal),
					fakeMarshaler,
					fakeValidator,
				)
			})
			It("wraps and returns an error", func() {
				_, err := mapper.AsBytes([]store.Policy{})
				Expect(err).To(MatchError(errors.New("marshal json: banana")))
			})
		})
	})

	Describe("MapStoreTag", func() {
		table.DescribeTable("should map store tags to api tags", func(input store.Tag, expected api.Tag) {
			result := api.MapStoreTag(input)
			Expect(result).To(Equal(expected))
		},
			table.Entry("direct translation",
				store.Tag{
					ID:   "some-id",
					Tag:  "some-tag",
					Type: "some-type",
				},
				api.Tag{
					ID:   "some-id",
					Tag:  "some-tag",
					Type: "some-type",
				},
			),
			table.Entry("direct translation",
				store.Tag{
					ID:   "some-other-id",
					Tag:  "some-other-tag",
					Type: "some-other-type",
				},
				api.Tag{
					ID:   "some-other-id",
					Tag:  "some-other-tag",
					Type: "some-other-type",
				},
			),
		)
	})

	Describe("MapStoreTags", func() {
		table.DescribeTable("should map store tags to api tags", func(input []store.Tag, expected []api.Tag) {
			result := api.MapStoreTags(input)
			Expect(result).To(Equal(expected))
		},
			table.Entry("direct translation",
				[]store.Tag{{
					ID:   "some-id",
					Tag:  "some-tag",
					Type: "some-type",
				}},
				[]api.Tag{{
					ID:   "some-id",
					Tag:  "some-tag",
					Type: "some-type",
				}},
			),
			table.Entry("direct translation",
				[]store.Tag{{
					ID:   "some-id",
					Tag:  "some-tag",
					Type: "some-type",
				},
					{
						ID:   "some-other-id",
						Tag:  "some-other-tag",
						Type: "some-other-type",
					}},
				[]api.Tag{{
					ID:   "some-id",
					Tag:  "some-tag",
					Type: "some-type",
				},
					{
						ID:   "some-other-id",
						Tag:  "some-other-tag",
						Type: "some-other-type",
					}},
			),
		)
	})
})
