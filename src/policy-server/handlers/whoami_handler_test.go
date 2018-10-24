package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Who Am I Handler", func() {
	var (
		request           *http.Request
		handler           *handlers.WhoAmIHandler
		resp              *httptest.ResponseRecorder
		logger            *lagertest.TestLogger
		expectedLogger    lager.Logger
		tokenData         uaa_client.CheckTokenResponse
		fakeErrorResponse *fakes.ErrorResponse
	)

	BeforeEach(func() {
		var err error
		request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
		Expect(err).NotTo(HaveOccurred())

		logger = lagertest.NewTestLogger("test")
		expectedLogger = lager.NewLogger("test").Session("who-am-i")

		testSink := lagertest.NewTestSink()
		expectedLogger.RegisterSink(testSink)
		expectedLogger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		fakeErrorResponse = &fakes.ErrorResponse{}
		handler = &handlers.WhoAmIHandler{
			Marshaler:     marshal.MarshalFunc(json.Marshal),
			ErrorResponse: fakeErrorResponse,
		}
		resp = httptest.NewRecorder()
		tokenData = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.admin"},
			UserName: "some_user",
		}
	})

	It("returns the username", func() {
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(Equal(`{"subject":"some_user"}`))
	})

	Context("when the subject token is a client", func() {
		It("returns the client id", func() {
			tokenData = uaa_client.CheckTokenResponse{
				Scope:   []string{"network.admin"},
				Subject: "some-client-id",
			}

			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(Equal(`{"subject":"some-client-id"}`))
		})
	})

	Context("when the logger isn't on the request context", func() {
		It("still works", func() {
			MakeRequestWithAuth(handler.ServeHTTP, resp, request, tokenData)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(Equal(`{"subject":"some_user"}`))
		})
	})

	Context("when the token data isn't on the request context", func() {
		It("still works", func() {
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(Equal(`{"subject":""}`))
		})
	})

	Context("when json marshaling the response fails", func() {
		BeforeEach(func() {
			handler.Marshaler = marshal.MarshalFunc(func(input interface{}) ([]byte, error) {
				return nil, errors.New("banana")
			})
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("marshaling response failed"))
		})
	})
})
