package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"

	apifakes "policy-server/api/fakes"

	"code.cloudfoundry.org/lager"

	"policy-server/store"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PoliciesIndexInternal", func() {
	var (
		handler              *handlers.PoliciesIndexInternal
		resp                 *httptest.ResponseRecorder
		fakeStore            *fakes.DataStore
		fakeErrorResponse    *fakes.ErrorResponse
		logger               *lagertest.TestLogger
		fakeMapper           *apifakes.PolicyMapper
		expectedResponseBody []byte
	)

	BeforeEach(func() {
		allPolicies := []store.Policy{{
			Source: store.Source{ID: "some-app-guid"},
			Destination: store.Destination{
				ID:       "some-other-app-guid",
				Protocol: "tcp",
				Ports: store.Ports{
					Start: 8080,
					End:   8080,
				},
			},
		}, {
			Source: store.Source{ID: "another-app-guid"},
			Destination: store.Destination{
				ID:       "some-other-app-guid",
				Protocol: "tcp",
				Ports: store.Ports{
					Start: 1234,
					End:   1234,
				},
			},
		},
		}

		byGuidsPolicies := []store.Policy{{
			Source: store.Source{ID: "some-app-guid"},
			Destination: store.Destination{
				ID:       "some-other-app-guid",
				Protocol: "tcp",
				Ports: store.Ports{
					Start: 8080,
					End:   8080,
				},
			},
		}}
		expectedResponseBody = []byte("some-response")

		fakeMapper = &apifakes.PolicyMapper{}
		fakeStore = &fakes.DataStore{}
		fakeStore.AllReturns(allPolicies, nil)
		fakeStore.ByGuidsReturns(byGuidsPolicies, nil)
		fakeMapper.AsBytesReturns(expectedResponseBody, nil)
		logger = lagertest.NewTestLogger("test")
		fakeErrorResponse = &fakes.ErrorResponse{}
		handler = &handlers.PoliciesIndexInternal{
			Logger:        logger,
			Store:         fakeStore,
			Mapper:        fakeMapper,
			ErrorResponse: fakeErrorResponse,
		}
		resp = httptest.NewRecorder()
	})

	It("it returns the policies returned by ByGuids", func() {
		request, err := http.NewRequest("GET", "/networking/v0/internal/policies?id=some-app-guid", nil)
		Expect(err).NotTo(HaveOccurred())
		MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

		Expect(fakeStore.ByGuidsCallCount()).To(Equal(1))
		srcGuids, dstGuids := fakeStore.ByGuidsArgsForCall(0)
		Expect(srcGuids).To(Equal([]string{"some-app-guid"}))
		Expect(dstGuids).To(Equal([]string{"some-app-guid"}))
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.Bytes()).To(Equal(expectedResponseBody))
	})

	Context("when the logger isn't on the request context", func() {
		It("still works", func() {
			request, err := http.NewRequest("GET", "/networking/v0/internal/policies?id=some-app-guid", nil)
			Expect(err).NotTo(HaveOccurred())
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.Bytes()).To(Equal(expectedResponseBody))
		})
	})

	Context("when there are policies and no ids are passed", func() {
		It("returns all of them", func() {
			request, err := http.NewRequest("GET", "/networking/v0/internal/policies", nil)
			Expect(err).NotTo(HaveOccurred())
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeStore.AllCallCount()).To(Equal(1))
			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.Bytes()).To(Equal(expectedResponseBody))
		})
	})

	Context("when rendering the policies as bytes fails", func() {
		BeforeEach(func() {
			fakeMapper.AsBytesReturns(nil, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			request, err := http.NewRequest("GET", "/networking/v0/internal/policies", nil)
			Expect(err).NotTo(HaveOccurred())
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-index-internal"))
			Expect(description).To(Equal("map policy as bytes failed"))

			By("logging the error")
			Expect(logger.Logs()).To(HaveLen(1))
			Expect(logger.Logs()[0]).To(SatisfyAll(
				LogsWith(lager.ERROR, "test.index-policies-internal.failed-mapping-policies-as-bytes"),
				HaveLogData(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("error", "banana"),
					HaveKeyWithValue("session", "1"),
				)),
			))
		})
	})

	Context("when the store throws an error", func() {

		BeforeEach(func() {
			fakeStore.AllReturns(nil, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			request, err := http.NewRequest("GET", "/networking/v0/internal/policies", nil)
			Expect(err).NotTo(HaveOccurred())
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-index-internal"))
			Expect(description).To(Equal("database read failed"))

			By("logging the error")
			Expect(logger.Logs()).To(HaveLen(1))
			Expect(logger.Logs()[0]).To(SatisfyAll(
				LogsWith(lager.ERROR, "test.index-policies-internal.failed-reading-database"),
				HaveLogData(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("error", "banana"),
					HaveKeyWithValue("session", "1"),
				)),
			))
		})
	})

})
