package handlers_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/fakes"
	"policy-server/handlers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("UaaHandler", func() {
	var (
		request   *http.Request
		handler   *handlers.WhoAmIHandler
		resp      *httptest.ResponseRecorder
		uaaClient *fakes.UAARequestClient
	)

	BeforeEach(func() {
		var err error
		request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
		Expect(err).NotTo(HaveOccurred())
		request.Header.Set("Authorization", "Bearer correct-token")

		uaaClient = &fakes.UAARequestClient{}
		handler = &handlers.WhoAmIHandler{
			Client: uaaClient,
		}
		uaaClient.GetNameReturns("some_user", nil)
		resp = httptest.NewRecorder()
	})

	It("checks the policies for the token from UAA", func() {
		handler.ServeHTTP(resp, request)

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(Equal(`{"user_name":"some_user"}`))
	})

	Context("when the header does not have a bearer token", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns a 403 status code", func() {
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Context("when the request client returns an error", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
			Expect(err).NotTo(HaveOccurred())
			uaaClient.GetNameReturns("", errors.New("potato"))
		})

		It("returns a 403 status code", func() {
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})
	})
})
