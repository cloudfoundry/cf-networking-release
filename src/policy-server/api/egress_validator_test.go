package api_test

import (
	"errors"
	"policy-server/api"
	"policy-server/api/fakes"
	"policy-server/store"

	"code.cloudfoundry.org/cf-networking-helpers/httperror"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Egress PolicyValidator", func() {
	var (
		validator        api.EgressValidator
		egressPolicies   []api.EgressPolicy
		ccClient         *fakes.CCClient
		uaaClient        *fakes.UAAClient
		destinationStore *fakes.EgressDestinationStore
	)

	BeforeEach(func() {
		ccClient = new(fakes.CCClient)
		uaaClient = new(fakes.UAAClient)
		destinationStore = new(fakes.EgressDestinationStore)
		validator = api.EgressValidator{
			CCClient:         ccClient,
			UAAClient:        uaaClient,
			DestinationStore: destinationStore,
		}
		ccClient.GetLiveAppGUIDsReturns(map[string]struct{}{
			"source-app-id": {},
			"source--id":    {},
		}, nil)

		ccClient.GetLiveSpaceGUIDsReturns(map[string]struct{}{
			"source-space-id": {},
		}, nil)

		uaaClient.GetTokenReturns("valid-token", nil)

		egressPolicies = []api.EgressPolicy{
			{
				Source: &api.EgressSource{
					ID: "source-app-id",
				},
				Destination: &api.EgressDestination{
					GUID: "existing-guid",
				},
				AppLifecycle: stringPtr("running"),
			},
			{
				Source: &api.EgressSource{
					ID: "source-app-id",
				},
				Destination: &api.EgressDestination{
					GUID: "existing-guid",
				},
				AppLifecycle: stringPtr("staging"),
			},
			{
				Source: &api.EgressSource{
					ID: "source-app-id",
				},
				Destination: &api.EgressDestination{
					GUID: "existing-guid",
				},
				AppLifecycle: stringPtr("all"),
			},
		}

		destinationStore.GetByGUIDReturns([]store.EgressDestination{{GUID: "existing-guid"}}, nil)
	})

	Describe("ValidateEgressPolicies", func() {
		Context("AppLifecycle", func(){
			It("returns an error when app_lifecycle is invalid", func(){
				egressPolicies[0].AppLifecycle = stringPtr("meow")
				err := validator.ValidateEgressPolicies(egressPolicies)
				Expect(err).To(MatchError(ContainSubstring("app lifecycle must be 'running', 'staging', or 'all'")))
			})
		})

		It("should not return an error when given a valid egress policy", func() {
			Expect(validator.ValidateEgressPolicies(egressPolicies)).To(Succeed())
		})

		It("requires a source", func() {
			egressPolicies[0].Source = nil

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError(ContainSubstring("missing egress source")))
		})

		It("requires the source app to exist", func() {
			egressPolicies = []api.EgressPolicy{
				{
					Source: &api.EgressSource{
						ID: "source-app-id",
					},
					Destination: &api.EgressDestination{
						GUID: "doesn't matter for this test",
					},
				},
				{
					Source: &api.EgressSource{
						ID: "non-existent",
					},
					Destination: &api.EgressDestination{
						GUID: "doesn't matter for this test",
					},
				},
				{
					Source: &api.EgressSource{
						ID:   "non-existent-2",
						Type: "app",
					},
					Destination: &api.EgressDestination{
						GUID: "doesn't matter for this test",
					},
				},
			}

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError(ContainSubstring("app guids not found: [non-existent, non-existent-2]")))

			egressPolicyError, ok := err.(httperror.MetadataError)
			Expect(ok).To(BeTrue(), "expected error to be of type MetadataError")
			Expect(egressPolicyError.Metadata()).To(Equal(map[string]interface{}{
				"policies with missing apps": egressPolicies[1:],
			}))

			Expect(uaaClient.GetTokenCallCount()).To(Equal(1))

			passedToken, passedAppGUIDs := ccClient.GetLiveAppGUIDsArgsForCall(0)
			Expect(passedToken).To(Equal("valid-token"))
			Expect(passedAppGUIDs).To(ConsistOf("source-app-id", "non-existent", "non-existent-2"))

			Expect(ccClient.GetLiveSpaceGUIDsCallCount()).To(Equal(0))
		})

		It("requires the destination to exist", func() {
			egressPolicies = []api.EgressPolicy{
				{
					Source: &api.EgressSource{
						ID: "source-app-id",
					},
					Destination: &api.EgressDestination{
						GUID: "existing-guid",
					},
				},
				{
					Source: &api.EgressSource{
						ID:   "source-app-id",
						Type: "app",
					},
					Destination: &api.EgressDestination{
						GUID: "non-existent-guid",
					},
				},
				{
					Source: &api.EgressSource{
						ID:   "source-app-id",
						Type: "app",
					},
					Destination: &api.EgressDestination{
						GUID: "non-existent-2",
					},
				},
			}

			destinationStore.GetByGUIDReturns([]store.EgressDestination{{GUID: "existing-guid"}}, nil)

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError(ContainSubstring("destination guids not found: [non-existent-2, non-existent-guid]")))

			egressPolicyError, ok := err.(httperror.MetadataError)
			Expect(ok).To(BeTrue(), "expected error to be of type MetadataError")
			Expect(egressPolicyError.Metadata()).To(Equal(map[string]interface{}{
				"policies with missing destinations": egressPolicies[1:],
			}))

			Expect(destinationStore.GetByGUIDArgsForCall(0)).To(ConsistOf("existing-guid", "non-existent-guid", "non-existent-2"))
		})

		It("returns an error if it can't query live app guids", func() {
			ccClient.GetLiveAppGUIDsReturns(nil, errors.New("foxtrot"))
			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError(ContainSubstring("failed to get live app guids: foxtrot")))
		})

		It("requires the source space to exist", func() {
			egressPolicies = []api.EgressPolicy{
				{
					Source: &api.EgressSource{
						ID:   "source-space-id",
						Type: "space",
					},
					Destination: &api.EgressDestination{
						GUID: "abc123",
					},
				},
				{
					Source: &api.EgressSource{
						ID:   "non-existent-space",
						Type: "space",
					},
					Destination: &api.EgressDestination{
						GUID: "def456",
					},
				},
				{
					Source: &api.EgressSource{
						ID:   "non-existent-space-2",
						Type: "space",
					},
					Destination: &api.EgressDestination{
						GUID: "ghi789",
					},
				},
			}

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError(ContainSubstring("space guids not found: [non-existent-space, non-existent-space-2]")))
			egressPolicyError, ok := err.(httperror.MetadataError)
			Expect(ok).To(BeTrue(), "expected error to be of type MetadataError")
			Expect(egressPolicyError.Metadata()).To(Equal(map[string]interface{}{
				"policies with missing spaces": egressPolicies[1:],
			}))


			Expect(uaaClient.GetTokenCallCount()).To(Equal(1))

			passedToken, passedSpaceGUIDs := ccClient.GetLiveSpaceGUIDsArgsForCall(0)
			Expect(passedToken).To(Equal("valid-token"))
			Expect(passedSpaceGUIDs).To(ConsistOf("source-space-id", "non-existent-space", "non-existent-space-2"))

			Expect(ccClient.GetLiveAppGUIDsCallCount()).To(Equal(0))
		})

		It("returns an error if it can't query live space guids", func() {
			egressPolicies[0].Source.Type = "space"

			ccClient.GetLiveSpaceGUIDsReturns(nil, errors.New("india"))
			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError(ContainSubstring("failed to get live space guids: india")))
		})

		It("returns an error if it can't query  destination guids", func() {
			destinationStore.GetByGUIDReturns(nil, errors.New("can't get destinations"))
			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError(ContainSubstring("failed to get egress destinations: can't get destinations")))
		})

		It("returns an error when it is unable to obtain a token", func() {
			uaaClient.GetTokenReturns("", errors.New("kilo"))

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError(ContainSubstring("failed to get uaa token: kilo")))
		})

		It("type must be app, space or empty", func() {
			egressPolicies[0].Source.Type = "invalid"

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError(ContainSubstring("source type must be app or space")))

			for _, validType := range []string{"app", "space", ""} {
				egressPolicies[0].Source.Type = validType
				egressPolicies[0].Source.ID = "source-" + validType + "-id"
				err := validator.ValidateEgressPolicies(egressPolicies)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("requires a source guid", func() {
			egressPolicies[0].Source.ID = ""

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError(ContainSubstring("missing egress source ID")))
		})

		It("requires a destination", func() {
			egressPolicies[0].Destination = nil

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError(ContainSubstring("missing egress destination")))
		})

		It("returns the bad record when validation fails", func() {
			egressPolicies = []api.EgressPolicy{
				{
					Source: &api.EgressSource{
						ID: "bad-record",
					},
					Destination: &api.EgressDestination{},
				},
			}

			err := validator.ValidateEgressPolicies(egressPolicies)
			egressPolicyError := err.(httperror.MetadataError)
			Expect(egressPolicyError.Metadata()).To(Equal(map[string]interface{}{"bad_egress_policy": egressPolicies[0]}))
		})
	})
})

func stringPtr(x string) *string {
	return &x
}
