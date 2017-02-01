package handlers_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Authentication middleware", func() {
	var (
		request       *http.Request
		unprotected   *fakes.AuthenticatedHandler
		protected     http.Handler
		authenticator *handlers.Authenticator

		resp          *httptest.ResponseRecorder
		uaaClient     *fakes.UAAClient
		logger        *lagertest.TestLogger
		tokenResponse uaa_client.CheckTokenResponse
	)

	BeforeEach(func() {
		var err error
		request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
		Expect(err).NotTo(HaveOccurred())
		request.Header.Set("Authorization", "Bearer correct-token")
		request.RemoteAddr = "some-host:some-ip"

		uaaClient = &fakes.UAAClient{}
		logger = lagertest.NewTestLogger("test")
		unprotected = &fakes.AuthenticatedHandler{}

		authenticator = &handlers.Authenticator{
			Client: uaaClient,
			Logger: logger,
			Scopes: []string{"network.admin", "network.write"},
		}

		protected = authenticator.Wrap(unprotected)

		tokenResponse = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.admin"},
			UserName: "some_user",
		}

		uaaClient.CheckTokenReturns(tokenResponse, nil)
		resp = httptest.NewRecorder()

	})

	It("calls into the unprotected handler and logs the request", func() {
		protected.ServeHTTP(resp, request)

		Expect(logger).To(gbytes.Say("request made to policy-server.*RemoteAddr.*some-host:some-ip.*URL.*/networking/v0/whoami"))
		Expect(unprotected.ServeHTTPCallCount()).To(Equal(1))

		Expect(logger).To(gbytes.Say("request made with token:.*tokenData.*scope.*network.admin.*user_name.*some_user"))
		unprotectedResp, unprotectedRequest, tokenData := unprotected.ServeHTTPArgsForCall(0)
		Expect(unprotectedResp).To(Equal(resp))
		Expect(unprotectedRequest).To(Equal(request))
		Expect(tokenData).To(Equal(tokenResponse))
	})

	It("checks the authorization bearer token with the uaa client", func() {
		protected.ServeHTTP(resp, request)

		Expect(uaaClient.CheckTokenCallCount()).To(Equal(1))
		Expect(uaaClient.CheckTokenArgsForCall(0)).To(Equal("correct-token"))
	})

	Context("when the header has a lowercase bearer token", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Authorization", "bearer correct-token")
		})

		It("checks the policies for the token from UAA", func() {
			protected.ServeHTTP(resp, request)

			Expect(uaaClient.CheckTokenCallCount()).To(Equal(1))
			Expect(uaaClient.CheckTokenArgsForCall(0)).To(Equal("correct-token"))
		})
	})

	Context("when the header does not have any authorization header", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns a 401 status code and a useful JSON error", func() {
			protected.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusUnauthorized))
			Expect(resp.Body).To(MatchJSON(`{ "error": "missing authorization header" }`))
		})

		It("logs the error", func() {
			protected.ServeHTTP(resp, request)

			Expect(logger).To(gbytes.Say("no auth header"))
		})
	})

	Context("when the request client returns an error", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Authorization", "Bearer incorrect-token")
			uaaClient.CheckTokenReturns(uaa_client.CheckTokenResponse{UserName: ""}, errors.New("potato"))
		})

		It("returns a 403 status code and a useful JSON error", func() {
			protected.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusForbidden))
			Expect(resp.Body).To(MatchJSON(`{ "error": "failed to verify token with uaa" }`))
		})

		It("logs the error", func() {
			protected.ServeHTTP(resp, request)

			Expect(logger).To(gbytes.Say("uaa-getname.*potato"))
		})
	})

	Context("when the token data scopes do not contain any of the allowed scopes", func() {
		BeforeEach(func() {
			uaaClient.CheckTokenReturns(uaa_client.CheckTokenResponse{
				Scope:    []string{"wrong.scope"},
				UserName: "some-user",
			}, nil)
		})

		It("returns a 403 status code and a useful JSON error", func() {
			protected.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusForbidden))
			Expect(resp.Body).To(MatchJSON(`{ "error": "token missing allowed scopes: [network.admin network.write]" }`))
		})

		It("logs a helpful error", func() {
			protected.ServeHTTP(resp, request)

			Expect(logger).To(gbytes.Say("allowed-scopes.*network.admin.*network.write"))
			Expect(logger).To(gbytes.Say("no allowed scopes found"))
			Expect(logger).To(gbytes.Say("provided-scopes.*wrong.scope"))
		})
	})
})
