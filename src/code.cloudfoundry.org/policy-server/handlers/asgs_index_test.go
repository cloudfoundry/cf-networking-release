package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	apifakes "code.cloudfoundry.org/policy-server/api/fakes"
	"code.cloudfoundry.org/policy-server/handlers"
	"code.cloudfoundry.org/policy-server/handlers/fakes"
	"code.cloudfoundry.org/policy-server/store"
	storeFakes "code.cloudfoundry.org/policy-server/store/fakes"
	"code.cloudfoundry.org/policy-server/uaa_client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Asgs per space index handler", func() {
	var (
		securityGroups       []store.SecurityGroup
		expectedResponseBody string
		request              *http.Request
		handler              *handlers.AsgsIndex
		resp                 *httptest.ResponseRecorder
		fakeStore            *storeFakes.SecurityGroupsStore
		fakeErrorResponse    *fakes.ErrorResponse
		fakeMapper           *apifakes.AsgMapper
		logger               *lagertest.TestLogger
		expectedLogger       lager.Logger
		token                uaa_client.CheckTokenResponse
	)

	BeforeEach(func() {
		securityGroups = []store.SecurityGroup{}

		var err error
		request, err = http.NewRequest("GET", "/networking/v1/external/security_group_rules", nil)
		Expect(err).NotTo(HaveOccurred())

		fakeStore = &storeFakes.SecurityGroupsStore{}
		fakeStore.BySpaceGuidsReturns(securityGroups, store.Pagination{}, nil)

		fakeErrorResponse = &fakes.ErrorResponse{}
		fakeMapper = &apifakes.AsgMapper{}

		logger = lagertest.NewTestLogger("test")
		handler = &handlers.AsgsIndex{
			Store:  fakeStore,
			Mapper: fakeMapper,
			// PolicyFilter:  fakePolicyFilter,
			// PolicyGuard:   fakePolicyGuard,
			ErrorResponse: fakeErrorResponse,
		}

		token = uaa_client.CheckTokenResponse{
			Scope: []string{"some-scope", "some-other-scope"},
		}
		resp = httptest.NewRecorder()

		expectedLogger = lager.NewLogger("test").Session("index-security-group-rules")

		testSink := lagertest.NewTestSink()
		expectedLogger.RegisterSink(testSink)
		expectedLogger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

		expectedResponseBody = `{
			"next": 0,
			"security_groupss": [
				{
					"id": 1,
					"guid": "sg1-guid",
					"name": "sg1",
				},
				{
					"id": 2,
					"guid": "sg2-guid"
					"name": "sg2",
				}
			]
		}`
		fakeMapper.AsBytesReturns([]byte(expectedResponseBody), nil)
	})

	Context("with no query params", func() {
		It("returns empty list returned by BySpaceGuids", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

			Expect(fakeStore.BySpaceGuidsCallCount()).To(Equal(1))
			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(Equal(expectedResponseBody))
			spaceGuids, page := fakeStore.BySpaceGuidsArgsForCall(0)
			Expect(spaceGuids).To(BeEmpty())
			Expect(page).To(Equal(store.Page{From: 0, Limit: 0}))
		})
	})

	Context("when from and limit parameters are passed in", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v1/external/security_group_rules?from=3&limit=2", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("queries store with Page argument", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

			Expect(fakeStore.BySpaceGuidsCallCount()).To(Equal(1))
			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(Equal(expectedResponseBody))
			_, page := fakeStore.BySpaceGuidsArgsForCall(0)
			Expect(page.Limit).To(Equal(2))
			Expect(page.From).To(Equal(3))
		})

	})

	Context("when a list of space guids is provided as a query parameter", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v1/external/security_group_rules?space_guids=space-a,space-b", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("filters on only those security groups returned by BySpaceGuids", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

			Expect(fakeStore.BySpaceGuidsCallCount()).To(Equal(1))
			spaceGuids, _ := fakeStore.BySpaceGuidsArgsForCall(0)
			Expect(spaceGuids).To(ConsistOf([]string{"space-a", "space-b"}))
			Expect(resp.Code).To(Equal(http.StatusOK))
		})
	})

	Context("when invalid from parameter is passed in", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v1/external/security_group_rules?from=something", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(HaveOccurred())
			Expect(description).To(Equal("invalid value for 'from' parameter"))
		})
	})

	Context("when invalid limit parameter is passed in", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v1/external/security_group_rules?limit=something", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(HaveOccurred())
			Expect(description).To(Equal("invalid value for 'limit' parameter"))
		})
	})

	Context("when the logger isn't on the request context", func() {
		It("still works", func() {
			MakeRequestWithAuth(handler.ServeHTTP, resp, request, token)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(Equal(expectedResponseBody))
		})
	})

	Context("when the store throws an error", func() {
		BeforeEach(func() {
			fakeStore.BySpaceGuidsReturns(nil, store.Pagination{}, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("database read failed"))
		})
	})

	Context("when mapping the asgs as bytes fails", func() {
		BeforeEach(func() {
			fakeMapper.AsBytesReturns(nil, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("map asgs as bytes failed"))
		})
	})
})
