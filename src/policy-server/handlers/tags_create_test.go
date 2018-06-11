package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"

	"bytes"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
)

var _ = Describe("Tags index handler", func() {
	var (
		request           *http.Request
		handler           *handlers.TagsCreate
		resp              *httptest.ResponseRecorder
		fakeStore         *fakes.CreateTagDataStore
		fakeErrorResponse *fakes.ErrorResponse
		logger            *lagertest.TestLogger
		expectedLogger    lager.Logger
		requestBody       string
		expectedGroupType string
		expectedGroupGuid string
	)

	BeforeEach(func() {
		expectedGroupType = "router-type"
		expectedGroupGuid = "router-guid"
		var err error
		requestBody = fmt.Sprintf(`{"type": "%s", "guid": "%s"}`, expectedGroupType, expectedGroupGuid)
		request, err = http.NewRequest("POST", "/networking/v0/tags", bytes.NewBuffer([]byte(requestBody)))
		Expect(err).NotTo(HaveOccurred())

		fakeStore = &fakes.CreateTagDataStore{}
		fakeErrorResponse = &fakes.ErrorResponse{}

		logger = lagertest.NewTestLogger("test")
		expectedLogger = lager.NewLogger("test").Session("create-tags")

		testSink := lagertest.NewTestSink()
		expectedLogger.RegisterSink(testSink)
		expectedLogger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		handler = &handlers.TagsCreate{
			Store:         fakeStore,
			ErrorResponse: fakeErrorResponse,
		}
		resp = httptest.NewRecorder()

		fakeStore.CreateTagReturns("0001", nil)
	})

	It("runs", func() {
		MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

		Expect(fakeStore.CreateTagCallCount()).To(Equal(1))
		Expect(resp.Code).To(Equal(http.StatusOK))

		groupGuid, groupType := fakeStore.CreateTagArgsForCall(0)
		Expect(groupGuid).To(Equal(expectedGroupGuid))
		Expect(groupType).To(Equal(expectedGroupType))

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(Equal(`{ "tag": "0001" }`))
	})

	Context("when there are errors reading the body bytes", func() {
		BeforeEach(func() {
			request.Body = ioutil.NopCloser(&testsupport.BadReader{})
		})

		It("calls the bad request handler", func() {
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeErrorResponse.BadRequestCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.BadRequestArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("failed reading request body"))
		})
	})

	Context("when the request json is invalid", func() {
		BeforeEach(func() {
			var err error
			requestBody = `{"BAD JSON}`
			request, err = http.NewRequest("POST", "/networking/v0/tags", bytes.NewBuffer([]byte(requestBody)))
			Expect(err).NotTo(HaveOccurred())
		})

		It("fails to parse and returns an error", func() {
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeErrorResponse.BadRequestCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.BadRequestArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("unexpected end of JSON input"))
			Expect(description).To(Equal("failed parsing request body"))
		})
	})

	Context("when CreateTag fails", func() {
		BeforeEach(func() {
			fakeStore.CreateTagReturns("", errors.New("meow meow"))
		})

		It("returns an error message", func() {
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("meow meow"))
			Expect(description).To(Equal("database create failed"))
		})
	})
})
