package handlers_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/cf-networking-helpers/middleware"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/policy-server/handlers"
	"code.cloudfoundry.org/policy-server/handlers/fakes"
	"code.cloudfoundry.org/policy-server/uaa_client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Authentication middleware", func() {
	var (
		request       *http.Request
		unprotected   http.HandlerFunc
		protected     http.Handler
		authenticator *handlers.Authenticator

		resp                 *httptest.ResponseRecorder
		uaaClient            *fakes.UAAClient
		logger               *lagertest.TestLogger
		expectedLogger       lager.Logger
		tokenResponse        uaa_client.CheckTokenResponse
		fakeErrorResponse    *fakes.ErrorResponse
		unprotectedCallCount = 0
	)

	BeforeEach(func() {
		var err error
		request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
		Expect(err).NotTo(HaveOccurred())
		request.Header.Set("Authorization", "Bearer correct-token")
		request.RemoteAddr = "some-host:some-ip"

		uaaClient = &fakes.UAAClient{}
		logger = lagertest.NewTestLogger("test")

		expectedLogger = lager.NewLogger("test").Session("authentication")

		testSink := lagertest.NewTestSink()
		expectedLogger.RegisterSink(testSink)
		expectedLogger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		unprotected = func(w http.ResponseWriter, r *http.Request) {
			unprotectedCallCount += 1
			By("passing the token data to the unprotected request")
			data := r.Context().Value(handlers.TokenDataKey)
			Expect(data).ToNot(BeNil())
			Expect(data).To(Equal(tokenResponse))

			Expect(w).To(Equal(resp))
		}
		unprotectedCallCount = 0
		fakeErrorResponse = &fakes.ErrorResponse{}

		authenticator = &handlers.Authenticator{
			Client:        uaaClient,
			Scopes:        []string{"network.admin", "network.write"},
			ErrorResponse: fakeErrorResponse,
			ScopeChecking: true,
		}

		protected = authenticator.Wrap(unprotected)

		tokenResponse = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.admin"},
			UserName: "some_user",
		}

		uaaClient.CheckTokenReturns(tokenResponse, nil)
		resp = httptest.NewRecorder()

	})

	makeRequest := func() {
		// Put logger on request
		if logger != nil {
			contextWithLogger := context.WithValue(request.Context(), middleware.Key("logger"), logger)
			request = request.WithContext(contextWithLogger)
		}
		protected.ServeHTTP(resp, request)

	}

	It("calls into the unprotected handler", func() {
		makeRequest()
		Expect(unprotectedCallCount).To(Equal(1))
	})

	It("checks the authorization bearer token with the uaa client", func() {
		makeRequest()
		Expect(unprotectedCallCount).To(Equal(1))

		Expect(uaaClient.CheckTokenCallCount()).To(Equal(1))
		Expect(uaaClient.CheckTokenArgsForCall(0)).To(Equal("correct-token"))
	})

	Context("when the logger isn't on the request", func() {
		BeforeEach(func() {
			logger = nil
		})
		It("still works", func() {
			makeRequest()
			Expect(unprotectedCallCount).To(Equal(1))
		})
	})

	Context("when we disable scope checking", func() {
		BeforeEach(func() {
			authenticator = &handlers.Authenticator{
				Client:        uaaClient,
				Scopes:        []string{"network.admin", "network.write"},
				ErrorResponse: fakeErrorResponse,
				ScopeChecking: false,
			}
			tokenResponse = uaa_client.CheckTokenResponse{
				Scope:    []string{},
				UserName: "some_user",
			}

			uaaClient.CheckTokenReturns(tokenResponse, nil)
			protected = authenticator.Wrap(unprotected)
		})
		It("calls the unprotected handler even when the token has no scopes", func() {
			makeRequest()
			Expect(unprotectedCallCount).To(Equal(1))
		})
	})

	Context("when the header has a lowercase bearer token", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Authorization", "bearer correct-token")
		})

		It("checks the policies for the token from UAA", func() {
			makeRequest()
			Expect(unprotectedCallCount).To(Equal(1))

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

		It("calls the unauthorized error handler", func() {
			makeRequest()
			Expect(unprotectedCallCount).To(Equal(0))

			Expect(fakeErrorResponse.UnauthorizedCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.UnauthorizedArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("no auth header"))
			Expect(description).To(Equal("missing authorization header"))
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

		It("calls the unauthorized error handler", func() {
			makeRequest()
			Expect(unprotectedCallCount).To(Equal(0))

			Expect(fakeErrorResponse.UnauthorizedCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.UnauthorizedArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("potato"))
			Expect(description).To(Equal("failed to verify token with uaa"))
		})
	})

	Context("when the token data scopes do not contain any of the allowed scopes", func() {
		BeforeEach(func() {
			uaaClient.CheckTokenReturns(uaa_client.CheckTokenResponse{
				Scope:    []string{"wrong.scope"},
				UserName: "some-user",
			}, nil)
		})

		It("calls the forbidden error handler", func() {
			makeRequest()
			Expect(unprotectedCallCount).To(Equal(0))

			Expect(fakeErrorResponse.ForbiddenCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.ForbiddenArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("provided scopes [wrong.scope] do not include allowed scopes [network.admin network.write]"))
			Expect(description).To(Equal("provided scopes [wrong.scope] do not include allowed scopes [network.admin network.write]"))
		})
	})
})
