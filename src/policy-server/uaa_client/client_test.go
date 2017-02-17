package uaa_client_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"policy-server/testsupport"
	"policy-server/uaa_client"
	"policy-server/uaa_client/fakes"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"code.cloudfoundry.org/lager/lagertest"
)

var _ = Describe("Client", func() {
	var (
		client           *uaa_client.Client
		httpClient       *fakes.HTTPClient
		returnedResponse *http.Response
		logger           *lagertest.TestLogger
	)

	Describe("GetToken", func() {
		BeforeEach(func() {
			httpClient = &fakes.HTTPClient{}
			logger = lagertest.NewTestLogger("test")
			client = &uaa_client.Client{
				BaseURL:    "https://some.base.url",
				Name:       "some-name",
				Secret:     "some-secret",
				HTTPClient: httpClient,
				Logger:     logger,
			}

			body := `
		{
  "access_token" : "valid-token",
  "token_type" : "bearer",
  "refresh_token" : "valid-token-r",
  "expires_in" : 43199,
  "scope" : "scim.userids openid cloud_controller.read password.write cloud_controller.write",
  "jti" : "9796365e7c364f41a9d2436aef6b8351"
}
		`
			returnedResponse = &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(body)),
			}
			httpClient.DoReturns(returnedResponse, nil)
		})

		It("Returns the token", func() {
			token, err := client.GetToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).To(Equal("valid-token"))

			receivedRequest := httpClient.DoArgsForCall(0)
			Expect(receivedRequest.Method).To(Equal("POST"))
			Expect(receivedRequest.URL.RawQuery).To(BeEmpty())
			receivedBytes, _ := ioutil.ReadAll(receivedRequest.Body)
			Expect(receivedBytes).To(Equal([]byte("client_id=some-name&grant_type=client_credentials")))

			authHeader := receivedRequest.Header["Authorization"]
			Expect(authHeader).To(HaveLen(1))
			Expect(authHeader[0]).To(Equal("Basic c29tZS1uYW1lOnNvbWUtc2VjcmV0"))

			contentType := receivedRequest.Header.Get("Content-Type")
			Expect(contentType).To(Equal("application/x-www-form-urlencoded"))
		})

		It("logs the request before sending", func() {
			_, err := client.GetToken()
			Expect(err).NotTo(HaveOccurred())

			Expect(logger).To(gbytes.Say("get-token"))
		})

		Context("when the http client returns an error", func() {
			BeforeEach(func() {
				httpClient.DoReturns(nil, errors.New("potato"))
				client.HTTPClient = httpClient
			})

			It("returns a helpful error", func() {
				_, err := client.GetToken()

				Expect(err).To(MatchError(ContainSubstring("http client: potato")))
			})
		})

		Context("if the response status code is not 200", func() {
			BeforeEach(func() {
				httpClient.DoReturns(&http.Response{
					StatusCode: 418,
					Body:       ioutil.NopCloser(strings.NewReader("bad thing")),
				}, nil)
				client.HTTPClient = httpClient
			})

			It("returns the response body in the error", func() {
				_, err := client.GetToken()

				Expect(err).To(Equal(uaa_client.BadUaaResponse{
					StatusCode:      418,
					UaaResponseBody: "bad thing",
				}))
			})
		})

		Context("when reading the body returns an error", func() {
			BeforeEach(func() {
				httpClient.DoReturns(&http.Response{Body: &testsupport.BadReader{}}, nil)
				client.HTTPClient = httpClient
			})

			It("returns a helpful error", func() {
				_, err := client.GetToken()

				Expect(err).To(MatchError(ContainSubstring("read body: banana")))
			})
		})

		Context("when the response body is not valid json", func() {
			BeforeEach(func() {
				returnedResponse = &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(`%%%%`)),
				}
				httpClient.DoReturns(returnedResponse, nil)
			})

			It("returns a helpful error", func() {
				_, err := client.GetToken()

				Expect(err).To(MatchError(ContainSubstring("unmarshal json: invalid character")))
			})
		})
	})

	Describe("CheckToken", func() {
		BeforeEach(func() {
			httpClient = &fakes.HTTPClient{}
			logger = lagertest.NewTestLogger("test")
			client = &uaa_client.Client{
				BaseURL:    "https://some.base.url",
				Name:       "test",
				Secret:     "test",
				HTTPClient: httpClient,
				Logger:     logger,
			}
			returnedResponse = &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(`{"scope":["network.admin"], "user_name":"some-user"}`)),
			}
			httpClient.DoReturns(returnedResponse, nil)
		})

		It("Returns the scopes and user name for the token", func() {
			fakeToken := fmt.Sprintf("%x", rand.Int31())
			tokenData, err := client.CheckToken(fakeToken)
			Expect(err).NotTo(HaveOccurred())

			receivedRequest := httpClient.DoArgsForCall(0)
			Expect(receivedRequest.Method).To(Equal("POST"))
			Expect(receivedRequest.URL.RawQuery).To(BeEmpty())
			receivedBytes, _ := ioutil.ReadAll(receivedRequest.Body)
			Expect(receivedBytes).To(Equal([]byte("token=" + fakeToken)))

			authHeader := receivedRequest.Header["Authorization"]
			Expect(authHeader).To(HaveLen(1))
			Expect(authHeader[0]).To(Equal("Basic dGVzdDp0ZXN0"))

			contentType := receivedRequest.Header.Get("Content-Type")
			Expect(contentType).To(Equal("application/x-www-form-urlencoded"))

			Expect(tokenData.UserName).To(Equal("some-user"))
			Expect(tokenData.Scope).To(Equal([]string{"network.admin"}))
		})

		It("logs the request before sending", func() {
			_, err := client.CheckToken("valid-token")
			Expect(err).NotTo(HaveOccurred())

			Expect(logger).To(gbytes.Say("check_token"))
		})

		Context("when the http client returns an error", func() {
			BeforeEach(func() {
				httpClient.DoReturns(nil, errors.New("potato"))
				client.HTTPClient = httpClient
			})

			It("returns a helpful error", func() {
				_, err := client.CheckToken("valid-token")

				Expect(err).To(MatchError(ContainSubstring("http client: potato")))
			})
		})

		Context("if the response status code is not 200", func() {
			BeforeEach(func() {
				httpClient.DoReturns(&http.Response{
					StatusCode: 418,
					Body:       ioutil.NopCloser(strings.NewReader("bad thing")),
				}, nil)
				client.HTTPClient = httpClient
			})

			It("returns the response body in the error", func() {
				_, err := client.CheckToken("something")

				Expect(err).To(Equal(uaa_client.BadUaaResponse{
					StatusCode:      418,
					UaaResponseBody: "bad thing",
				}))
			})
		})

		Context("when reading the body returns an error", func() {
			BeforeEach(func() {
				httpClient.DoReturns(&http.Response{Body: &testsupport.BadReader{}}, nil)
				client.HTTPClient = httpClient
			})

			It("returns a helpful error", func() {
				_, err := client.CheckToken("valid-token")

				Expect(err).To(MatchError(ContainSubstring("read body: banana")))
			})
		})

		Context("when the response body is not valid json", func() {
			BeforeEach(func() {
				returnedResponse = &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(`%%%%`)),
				}
				httpClient.DoReturns(returnedResponse, nil)
			})

			It("returns a helpful error", func() {
				_, err := client.CheckToken("valid-token")

				Expect(err).To(MatchError(ContainSubstring("unmarshal json: invalid character")))
			})
		})
	})
})
