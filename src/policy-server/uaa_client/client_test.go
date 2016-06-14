package uaa_client_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"policy-server/fakes"
	"policy-server/testsupport"
	"policy-server/uaa_client"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Client", func() {
	var (
		client           *uaa_client.Client
		httpClient       *fakes.HTTPClient
		returnedResponse *http.Response
		logger           *lagertest.TestLogger
	)

	Describe("GetName", func() {
		BeforeEach(func() {
			httpClient = &fakes.HTTPClient{}
			logger = lagertest.NewTestLogger("test")
			client = &uaa_client.Client{
				Host:       "some.url",
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

		It("Gets the username by posting to check_token uaa endpoint", func() {
			fakeToken := fmt.Sprintf("%x", rand.Int31())
			userName, err := client.GetName(fakeToken)
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

			Expect(userName).To(Equal("some-user"))
		})

		It("logs the request before sending", func() {
			_, err := client.GetName("valid-token")
			Expect(err).NotTo(HaveOccurred())

			Expect(logger).To(gbytes.Say("check_token"))
		})

		Context("when the http client returns an error", func() {
			BeforeEach(func() {
				httpClient.DoReturns(nil, errors.New("potato"))
				client.HTTPClient = httpClient
			})

			It("returns a helpful error", func() {
				_, err := client.GetName("valid-token")

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
				_, err := client.GetName("something")

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
				_, err := client.GetName("valid-token")

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
				_, err := client.GetName("valid-token")

				Expect(err).To(MatchError(ContainSubstring("unmarshal json: invalid character")))
			})
		})

		Context("when the response does not have the network.admin scope", func() {
			BeforeEach(func() {
				returnedResponse = &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(`{"scope":["wrong.scope"], "user_name":"some-user"}`)),
				}
				httpClient.DoReturns(returnedResponse, nil)
			})
			It("returns a helpful error", func() {
				_, err := client.GetName("valid-token")

				Expect(err).To(MatchError("network.admin scope not found"))
			})
		})
	})
})
