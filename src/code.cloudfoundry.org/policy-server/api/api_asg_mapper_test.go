package api

import (
	"encoding/json"
	"errors"

	hfakes "code.cloudfoundry.org/cf-networking-helpers/fakes"
	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/policy-server/store"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ApiAsgMapper", func() {

	var (
		mapper        AsgMapper
		fakeMarshaler *hfakes.Marshaler
	)
	BeforeEach(func() {
		mapper = NewAsgMapper(
			marshal.MarshalFunc(json.Marshal),
		)
		fakeMarshaler = &hfakes.Marshaler{}
	})

	Describe("AsBytes", func() {
		It("maps a slice of store.SecurityGroup to a payload with api.SecurityGroup", func() {
			securityGroups := []store.SecurityGroup{
				{
					Guid:           "sg1-guid",
					Name:           "sg1",
					Rules:          `["some-json"]`,
					StagingDefault: true,
					RunningDefault: false,
					StagingSpaceGuids: store.SpaceGuids{
						"space-a",
						"space-b",
					},
					RunningSpaceGuids: store.SpaceGuids{
						"space-a",
						"space-b",
					},
				}, {
					Guid:           "sg2-guid",
					Name:           "sg2",
					Rules:          `["some-other-json"]`,
					StagingDefault: true,
					RunningDefault: false,
					StagingSpaceGuids: store.SpaceGuids{
						"space-a",
						"space-b",
					},
					RunningSpaceGuids: store.SpaceGuids{
						"space-a",
						"space-b",
					},
				},
			}

			pagination := store.Pagination{
				Next: 3,
			}

			payload, err := mapper.AsBytes(securityGroups, pagination)
			Expect(err).NotTo(HaveOccurred())
			Expect(payload).To(MatchJSON(
				[]byte(`{
					"next": 3,
					"security_groups": [{
						"guid": "sg1-guid",
						"name": "sg1",
						"rules": "[\"some-json\"]",
						"staging_default": true,
						"running_default": false,
						"staging_space_guids": ["space-a", "space-b"],
						"running_space_guids": ["space-a", "space-b"]
					}, {
						"guid": "sg2-guid",
						"name": "sg2",
						"rules": "[\"some-other-json\"]",
						"staging_default": true,
						"running_default": false,
						"staging_space_guids": ["space-a", "space-b"],
						"running_space_guids": ["space-a", "space-b"]
					}]
				}`),
			))
		})

		Context("when marshalling fails", func() {
			BeforeEach(func() {
				fakeMarshaler.MarshalReturns(nil, errors.New("banana"))
				mapper = NewAsgMapper(
					fakeMarshaler,
				)
			})
			It("wraps and returns an error", func() {
				_, err := mapper.AsBytes([]store.SecurityGroup{}, store.Pagination{})
				Expect(err).To(MatchError(errors.New("marshal json: banana")))
			})
		})
	})
})
