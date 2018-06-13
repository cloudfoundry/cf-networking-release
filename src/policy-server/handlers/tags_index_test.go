package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/store"

	hfakes "code.cloudfoundry.org/cf-networking-helpers/fakes"
	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tags index handler", func() {
	var (
		allTags           []store.Tag
		request           *http.Request
		handler           *handlers.TagsIndex
		resp              *httptest.ResponseRecorder
		fakeStore         *fakes.DataStore
		fakeErrorResponse *fakes.ErrorResponse
		logger            *lagertest.TestLogger
		expectedLogger    lager.Logger
		marshaler         *hfakes.Marshaler
	)

	BeforeEach(func() {
		allTags = []store.Tag{{
			ID:  "some-app-guid",
			Tag: "0001",
			Type: "app",
		}, {
			ID:  "some-other-app-guid",
			Tag: "0002",
			Type: "app",
		}}

		var err error
		request, err = http.NewRequest("GET", "/networking/v0/external/tags", nil)
		Expect(err).NotTo(HaveOccurred())

		marshaler = &hfakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal

		fakeStore = &fakes.DataStore{}
		fakeErrorResponse = &fakes.ErrorResponse{}
		fakeStore.TagsReturns(allTags, nil)
		logger = lagertest.NewTestLogger("test")
		expectedLogger = lager.NewLogger("test").Session("index-tags")

		testSink := lagertest.NewTestSink()
		expectedLogger.RegisterSink(testSink)
		expectedLogger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		handler = &handlers.TagsIndex{
			Store:         fakeStore,
			Marshaler:     marshaler,
			ErrorResponse: fakeErrorResponse,
		}
		resp = httptest.NewRecorder()
	})

	It("returns all the tags", func() {
		expectedResponseJSON := `{"tags": [
			{ "id": "some-app-guid", "tag": "0001", "type": "app" },
			{ "id": "some-other-app-guid", "tag": "0002", "type": "app" }
        ]}`
		MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

		Expect(fakeStore.TagsCallCount()).To(Equal(1))
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body).To(MatchJSON(expectedResponseJSON))
	})

	Context("when the logger isn't on the request context", func() {
		BeforeEach(func() {
			logger = nil
		})
		It("still works", func() {
			expectedResponseJSON := `{"tags": [
			{ "id": "some-app-guid", "tag": "0001", "type": "app" },
			{ "id": "some-other-app-guid", "tag": "0002", "type": "app" }
        ]}`
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body).To(MatchJSON(expectedResponseJSON))
		})
	})

	Context("when the store throws an error", func() {
		BeforeEach(func() {
			fakeStore.TagsReturns(nil, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("database read failed"))
		})
	})

	Context("when the tags cannot be marshaled", func() {
		BeforeEach(func() {
			marshaler.MarshalStub = func(interface{}) ([]byte, error) {
				return nil, errors.New("grapes")
			}
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("grapes"))
			Expect(description).To(Equal("database marshalling failed"))
		})
	})
})
