package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"lib/marshal"
	"net/http"
	"net/http/httptest"
	"policy-server/fakes"
	"policy-server/handlers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("UaaHandler", func() {
	var (
		request   *http.Request
		handler   *handlers.WhoAmIHandler
		resp      *httptest.ResponseRecorder
		uaaClient *fakes.UAARequestClient
		logger    *lagertest.TestLogger
	)

	BeforeEach(func() {
		var err error
		request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
		Expect(err).NotTo(HaveOccurred())
		request.Header.Set("Authorization", "Bearer correct-token")

		uaaClient = &fakes.UAARequestClient{}
		logger = lagertest.NewTestLogger("test")
		handler = &handlers.WhoAmIHandler{
			Client:    uaaClient,
			Logger:    logger,
			Marshaler: marshal.MarshalFunc(json.Marshal),
		}
		uaaClient.GetNameReturns("some_user", nil)
		resp = httptest.NewRecorder()
	})

	It("checks the policies for the token from UAA", func() {
		handler.ServeHTTP(resp, request)

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(Equal(`{"user_name":"some_user"}`))
	})

	It("passes the token into the client", func() {
		handler.ServeHTTP(resp, request)

		Expect(uaaClient.GetNameCallCount()).To(Equal(1))
		Expect(uaaClient.GetNameArgsForCall(0)).To(Equal("correct-token"))
	})

	Context("when the header has a lowercase bearer token", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Authorization", "bearer correct-token")
		})

		It("checks the policies for the token from UAA", func() {
			handler.ServeHTTP(resp, request)

			Expect(uaaClient.GetNameCallCount()).To(Equal(1))
			Expect(uaaClient.GetNameArgsForCall(0)).To(Equal("correct-token"))
		})
	})

	Context("when the header does not have any authorization header", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns a 403 status code", func() {
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})

		It("logs the error", func() {
			handler.ServeHTTP(resp, request)

			Expect(logger).To(gbytes.Say("no auth header"))
		})
	})

	Context("when the request client returns an error", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Authorization", "Bearer incorrect-token")
			uaaClient.GetNameReturns("", errors.New("potato"))
		})

		It("returns a 403 status code", func() {
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})

		It("logs the error", func() {
			handler.ServeHTTP(resp, request)

			Expect(logger).To(gbytes.Say("uaa-getname.*potato"))
		})
	})

	Context("when json marshaling the response fails", func() {
		BeforeEach(func() {
			handler.Marshaler = marshal.MarshalFunc(func(input interface{}) ([]byte, error) {
				return nil, errors.New("banana")
			})
		})

		It("returns a 500 status code", func() {
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		})

		It("logs the error", func() {
			handler.ServeHTTP(resp, request)

			Expect(logger).To(gbytes.Say("marshal-response.*banana"))
		})

	})
})
