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

	"code.cloudfoundry.org/go-db-helpers/marshal"
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
		tokenData         uaa_client.CheckTokenResponse
		fakeErrorResponse *fakes.ErrorResponse
	)

	BeforeEach(func() {
		var err error
		request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
		Expect(err).NotTo(HaveOccurred())

		logger = lagertest.NewTestLogger("test")
		fakeErrorResponse = &fakes.ErrorResponse{}
		handler = &handlers.WhoAmIHandler{
			Logger:        logger,
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
		handler.ServeHTTP(resp, request, tokenData)

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(Equal(`{"user_name":"some_user"}`))
	})

	Context("when json marshaling the response fails", func() {
		BeforeEach(func() {
			handler.Marshaler = marshal.MarshalFunc(func(input interface{}) ([]byte, error) {
				return nil, errors.New("banana")
			})
		})

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("who-am-i"))
			Expect(description).To(Equal("marshaling response failed"))
		})
	})
})
