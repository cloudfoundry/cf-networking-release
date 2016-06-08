package uaa_client_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"policy-server/fakes"
	"policy-server/testsupport"
	"policy-server/uaa_client"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		client           *uaa_client.Client
		httpClient       *fakes.HTTPClient
		returnedResponse *http.Response
	)

	Describe("GetName", func() {
		BeforeEach(func() {
			httpClient = &fakes.HTTPClient{}
			client = &uaa_client.Client{
				Host:       "some.url",
				Name:       "test",
				Secret:     "test",
				HTTPClient: httpClient,
			}
			returnedResponse = &http.Response{
				Body: ioutil.NopCloser(strings.NewReader(`{"scope":["network.admin"], "user_name":"some-user"}`)),
			}
			httpClient.DoReturns(returnedResponse, nil)
		})

		It("Gets the username by posting to check_token uaa endpoint", func() {
			userName, err := client.GetName("valid-token")
			Expect(err).NotTo(HaveOccurred())

			receivedRequest := httpClient.DoArgsForCall(0)
			receivedBody, err := ioutil.ReadAll(receivedRequest.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(receivedRequest.Method).To(Equal("POST"))
			Expect(receivedBody).To(ContainSubstring("token=valid-token"))

			authHeader := receivedRequest.Header["Authorization"]
			Expect(authHeader).To(HaveLen(1))
			Expect(authHeader[0]).To(Equal("Basic dGVzdDp0ZXN0"))

			Expect(userName).To(Equal("some-user"))
		})

		Context("when the response body is not valid json", func() {

			BeforeEach(func() {
				returnedResponse = &http.Response{
					Body: ioutil.NopCloser(strings.NewReader(`%%%%`)),
				}
				httpClient.DoReturns(returnedResponse, nil)
			})

			It("returns a helpful error", func() {
				_, err := client.GetName("valid-token")

				Expect(err).To(MatchError(ContainSubstring("unmarshal json: invalid character")))
			})
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
	})
})
