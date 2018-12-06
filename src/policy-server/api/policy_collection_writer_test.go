package api_test

import (
	"encoding/json"
	"errors"
	"policy-server/api"
	"policy-server/store"

	hfakes "code.cloudfoundry.org/cf-networking-helpers/fakes"
	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyCollectionWriter", func() {
	var (
		writer        api.PolicyCollectionWriter
		fakeMarshaler *hfakes.Marshaler
	)
	BeforeEach(func() {
		writer = api.NewPolicyCollectionWriter(marshal.MarshalFunc(json.Marshal))
		fakeMarshaler = &hfakes.Marshaler{}
	})
	Describe("AsBytes", func() {
		It("maps a slice of store.Policy and store.EgressPolicy to a payload", func() {
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

			egressPolicies := []store.EgressPolicy{{
				Source: store.EgressSource{ID: "some-egress-app-guid", Type: "app"},
				Destination: store.EgressDestination{
					Rules: []store.EgressDestinationRule{
						{
							Protocol: "tcp",
							IPRanges: []store.IPRange{{Start: "8.0.8.0", End: "8.0.8.0"}},
							Ports:    []store.Ports{{Start: 80, End: 90}},
						},
						{
							Protocol: "icmp",
							IPRanges: []store.IPRange{{Start: "1.1.1.1", End: "2.2.2.2"}},
							ICMPType: 8,
							ICMPCode: 4,
						},
					},
				},
				AppLifecycle: "running",
			}}

			payload, err := writer.AsBytes(policies, egressPolicies)
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
					}],
					"total_egress_policies": 2,
					"egress_policies": [
						{
							"source": {"id": "some-egress-app-guid", "type": "app"},
							"destination": {
								"ips": [{"start": "8.0.8.0", "end": "8.0.8.0"}],
								"protocol": "tcp",
								"ports": [{"start": 80, "end": 90}]
							},
							"app_lifecycle": "running"
						},
						{
							"source": {"id": "some-egress-app-guid", "type": "app"},
							"destination": {
								"ips": [{"start": "1.1.1.1", "end": "2.2.2.2"}],
								"protocol": "icmp",
								"icmp_type": 8,
								"icmp_code": 4
							},
							"app_lifecycle": "running"
						}
					]
				}`),
			))
		})

		Context("when marshalling fails", func() {
			BeforeEach(func() {
				fakeMarshaler.MarshalReturns(nil, errors.New("banana"))
				writer = api.NewPolicyCollectionWriter(fakeMarshaler)
			})

			It("wraps and returns an error", func() {
				_, err := writer.AsBytes([]store.Policy{}, []store.EgressPolicy{})
				Expect(err).To(MatchError(errors.New("marshal json: banana")))
			})
		})
	})
})
