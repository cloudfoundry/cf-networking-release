package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"lib/marshal"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("UaaHandler", func() {
	var (
		request   *http.Request
		handler   *handlers.WhoAmIHandler
		resp      *httptest.ResponseRecorder
		logger    *lagertest.TestLogger
		tokenData uaa_client.CheckTokenResponse
	)

	BeforeEach(func() {
		var err error
		request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
		Expect(err).NotTo(HaveOccurred())

		logger = lagertest.NewTestLogger("test")
		handler = &handlers.WhoAmIHandler{
			Logger:    logger,
			Marshaler: marshal.MarshalFunc(json.Marshal),
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

		It("returns a 500 status code", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		})

		It("logs the error", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(logger).To(gbytes.Say("marshal-response.*banana"))
		})

	})
})
