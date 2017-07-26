package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/store"
	"policy-server/uaa_client"

	apifakes "policy-server/api/fakes"

	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PoliciesCleanup", func() {
	var (
		request           *http.Request
		handler           *handlers.PoliciesCleanup
		resp              *httptest.ResponseRecorder
		logger            *lagertest.TestLogger
		fakePolicyCleaner *fakes.PolicyCleaner
		fakeMapper        *apifakes.PolicyMapper
		fakeErrorResponse *fakes.ErrorResponse
		policies          []store.Policy
		tokenData         uaa_client.CheckTokenResponse
	)

	BeforeEach(func() {
		policies = []store.Policy{{
			Source: store.Source{ID: "live-guid", Tag: "tag"},
			Destination: store.Destination{
				ID:       "dead-guid",
				Tag:      "tag",
				Protocol: "tcp",
				Ports: store.Ports{
					Start: 8080,
					End:   8080,
				},
			},
		}}

		logger = lagertest.NewTestLogger("test")

		fakeMapper = &apifakes.PolicyMapper{}
		fakePolicyCleaner = &fakes.PolicyCleaner{}
		fakeErrorResponse = &fakes.ErrorResponse{}

		handler = &handlers.PoliciesCleanup{
			Mapper:        fakeMapper,
			PolicyCleaner: fakePolicyCleaner,
			ErrorResponse: fakeErrorResponse,
		}

		tokenData = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.admin"},
			UserName: "some_user",
		}

		fakePolicyCleaner.DeleteStalePoliciesReturns(policies, nil)
		fakeMapper.AsBytesReturns([]byte("some-bytes"), nil)
		resp = httptest.NewRecorder()
		request, _ = http.NewRequest("POST", "/networking/v0/external/policies/cleanup", nil)
	})

	It("Cleans up stale policies for deleted apps", func() {
		handler.ServeHTTP(logger, resp, request, tokenData)

		Expect(fakePolicyCleaner.DeleteStalePoliciesCallCount()).To(Equal(1))
		Expect(fakeMapper.AsBytesCallCount()).To(Equal(1))

		Expect(fakeMapper.AsBytesArgsForCall(0)).To(Equal(policies))

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(Equal(`some-bytes`))
	})

	Context("When deleting the policies fails", func() {
		BeforeEach(func() {
			fakePolicyCleaner.DeleteStalePoliciesReturns(nil, errors.New("potato"))
		})

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(logger, resp, request, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("potato"))
			Expect(message).To(Equal("policies-cleanup"))
			Expect(description).To(Equal("policies cleanup failed"))

			By("logging the error")
			Expect(logger.Logs()).To(HaveLen(1))
			Expect(logger.Logs()[0]).To(SatisfyAll(
				LogsWith(lager.ERROR, "test.cleanup-policies.failed-deleting-stale-policies"),
				HaveLogData(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("error", "potato"),
					HaveKeyWithValue("session", "1"),
				)),
			))
		})
	})

	Context("When mapping the policies to bytes", func() {
		BeforeEach(func() {
			fakeMapper.AsBytesReturns(nil, errors.New("potato"))
		})

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(logger, resp, request, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("potato"))
			Expect(message).To(Equal("policies-cleanup"))
			Expect(description).To(Equal("map policy as bytes failed"))

			By("logging the error")
			Expect(logger.Logs()).To(HaveLen(1))
			Expect(logger.Logs()[0]).To(SatisfyAll(
				LogsWith(lager.ERROR, "test.cleanup-policies.failed-mapping-policies-as-bytes"),
				HaveLogData(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("error", "potato"),
					HaveKeyWithValue("session", "1"),
				)),
			))
		})
	})
})
