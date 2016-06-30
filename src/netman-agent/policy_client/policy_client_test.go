package policy_client_test

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"netman-agent/models"
	"netman-agent/policy_client"
	"policy-server/fakes"
	"strings"

	lfakes "lib/fakes"
	"lib/testsupport"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("PolicyClient", func() {
	var (
		client           *policy_client.Client
		httpClient       *fakes.HTTPClient
		fakeUnmarshaler  *lfakes.Unmarshaler
		returnedResponse *http.Response
		logger           *lagertest.TestLogger
	)

	BeforeEach(func() {
		httpClient = &fakes.HTTPClient{}
		fakeUnmarshaler = &lfakes.Unmarshaler{}
		fakeUnmarshaler.UnmarshalStub = json.Unmarshal
		logger = lagertest.NewTestLogger("test")
		client = policy_client.New(logger, httpClient, "http://some.url", fakeUnmarshaler)

		returnedResponse = &http.Response{
			StatusCode: 200,
			Body: ioutil.NopCloser(strings.NewReader(`{
				"policies": [
				{
					"source": {
						"id": "some-app-guid"
					},
					"destination": {
						"id": "some-other-app-guid",
						"port": 5555,
						"protocol": "udp"
					}
				}
				]	
			}`)),
		}
		httpClient.DoReturns(returnedResponse, nil)
	})
	Describe("GetPolicies", func() {
		It("gets the policy list from the server, logs it and returns it", func() {
			policies, err := client.GetPolicies()
			Expect(err).NotTo(HaveOccurred())

			Expect(httpClient.DoCallCount()).To(Equal(1))
			receivedRequest := httpClient.DoArgsForCall(0)
			Expect(receivedRequest.Method).To(Equal("GET"))
			Expect(receivedRequest.URL.Host).To(Equal("some.url"))
			Expect(receivedRequest.URL.Path).To(Equal("/networking/v0/internal/policies"))

			Expect(policies).To(Equal([]models.Policy{
				{
					Source: models.Source{
						ID: "some-app-guid",
					},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Port:     5555,
						Protocol: "udp",
					},
				},
			},
			))
			Expect(logger).To(gbytes.Say(`get-policies.*some-app-guid.*some-other-app-guid`))
		})

		Context("when the request returns a non 200 status code", func() {
			BeforeEach(func() {
				returnedResponse.Body = ioutil.NopCloser(strings.NewReader(`{"error":"some-error"}`))
				returnedResponse.StatusCode = http.StatusBadRequest
				httpClient.DoReturns(returnedResponse, nil)
			})

			It("returns the error and logs the body", func() {
				_, err := client.GetPolicies()
				Expect(err).To(MatchError("http client do: bad response status 400"))

				Expect(logger).To(gbytes.Say(`http-client.*some-error.*400`))
			})
		})

		Context("when the json unmarshal fails", func() {
			BeforeEach(func() {
				fakeUnmarshaler.UnmarshalStub = func([]byte, interface{}) error {
					return errors.New("grapes")
				}
			})
			It("returns and logs the error", func() {
				_, err := client.GetPolicies()
				Expect(err).To(MatchError("json unmarshal: grapes"))
			})
		})

		Context("when the request creation fails", func() {
			BeforeEach(func() {
				client = policy_client.New(logger, httpClient, "%%%", fakeUnmarshaler)

			})
			It("returns and logs the error", func() {
				_, err := client.GetPolicies()
				Expect(err).To(MatchError(ContainSubstring(`invalid URL escape "%%%"`)))
			})
		})

		Context("when the request fails", func() {
			BeforeEach(func() {
				httpClient.DoReturns(nil, errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := client.GetPolicies()
				Expect(err).To(MatchError("http client do: banana"))
			})
		})

		Context("when the body read fails", func() {
			BeforeEach(func() {
				returnedResponse = &http.Response{
					StatusCode: 200,
					Body:       &testsupport.BadReader{},
				}
				httpClient.DoReturns(returnedResponse, nil)
			})
			It("returns the error", func() {
				_, err := client.GetPolicies()
				Expect(err).To(MatchError("body read: banana"))
			})
		})
	})
})
