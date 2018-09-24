package api_test

import (
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"policy-server/api"
	"policy-server/api/fakes"
)

var _ = Describe("PayloadValidator", func() {

	var (
		validator                      *api.PayloadValidator
		policyPayload                  *api.PoliciesPayload
		destinationsPayload             *api.DestinationsPayload
		fakePolicyValidator            *fakes.Validator
		fakeEgressDestinationValidator *fakes.EgressDestinationsValidator
	)
	BeforeEach(func() {
		fakePolicyValidator = &fakes.Validator{}
		fakeEgressDestinationValidator = &fakes.EgressDestinationsValidator{}
		fakeEgressDestinationValidator.ValidateEgressDestinationsReturns(nil)

		validator = &api.PayloadValidator{
			PolicyValidator:            fakePolicyValidator,
			EgressDestinationValidator: fakeEgressDestinationValidator,
		}

		policyPayload = &api.PoliciesPayload{
			Policies: []api.Policy{
				{},
			},
		}

		destinationsPayload = &api.DestinationsPayload{
			EgressDestinations: []api.EgressDestination{
				{},
			},
		}

	})

	Context("ValidatePayload", func() {
		It("returns no error if the payload is valid", func() {
			Expect(validator.ValidatePayload(policyPayload)).To(Succeed())
		})

		It("returns an error if both policy and egress policies are empty", func() {
			err := validator.ValidatePayload(&api.PoliciesPayload{})
			Expect(err).To(MatchError("expected policy"))
		})

		It("returns delegates to the policy validator and the egress policy validator", func() {
			validator.ValidatePayload(policyPayload)
			Expect(fakePolicyValidator.ValidatePoliciesCallCount()).To(Equal(1))
			Expect(fakePolicyValidator.ValidatePoliciesArgsForCall(0)).To(Equal(policyPayload.Policies))
		})

		It("returns an error if the policy validator returns an error", func() {
			fakePolicyValidator.ValidatePoliciesReturns(errors.New("policy validator error"))
			Expect(validator.ValidatePayload(policyPayload)).To(MatchError("policy validator error"))
		})

		It("does not invoke policy validator if policies are empty", func() {
			policyPayload.Policies = []api.Policy{}
			validator.ValidatePayload(policyPayload)
			Expect(fakePolicyValidator.ValidatePoliciesCallCount()).To(Equal(0))
		})
	})

	Context("ValidateEgressDestinationPayload", func() {
		It("returns no error if the payload is valid", func() {
			Expect(validator.ValidateEgressDestinationsPayload(destinationsPayload)).To(Succeed())
		})

		Context("when the policy is invalid", func(){
			BeforeEach(func(){
				fakeEgressDestinationValidator.ValidateEgressDestinationsReturns(errors.New("banana"))
			})

			It("returns an error", func(){
				Expect(validator.ValidateEgressDestinationsPayload(destinationsPayload)).To(MatchError("banana"))
			})
		})
	})
})
