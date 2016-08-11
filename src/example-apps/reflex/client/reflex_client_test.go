package client_test

import (
	"errors"
	"example-apps/reflex/client"
	"example-apps/reflex/fakes"
	"io/ioutil"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ReflexClient", func() {
	var (
		reflexClient *client.ReflexClient
		httpClient   *fakes.HttpClient
		resp         *http.Response
	)
	BeforeEach(func() {
		resp = &http.Response{}
		resp.Body = ioutil.NopCloser(strings.NewReader(`{"ips":["2.2.2.2","3.3.3.3"]}`))
		httpClient = &fakes.HttpClient{}
		httpClient.GetReturns(resp, nil)

		reflexClient = &client.ReflexClient{
			HttpClient: httpClient,
			AppURL:     "something",
			AppPort:    1234,
		}
	})
	Describe("GetAddressesViaRouter", func() {
		It("gets a response from the app and returns a list of addresses", func() {
			addresses, err := reflexClient.GetAddressesViaRouter()
			Expect(err).NotTo(HaveOccurred())

			Expect(httpClient.GetCallCount()).To(Equal(1))
			Expect(httpClient.GetArgsForCall(0)).To(Equal("http://something/peers"))
			Expect(addresses).To(ConsistOf("2.2.2.2", "3.3.3.3"))
		})

		Context("when the http client returns an error", func() {
			BeforeEach(func() {
				httpClient.GetReturns(nil, errors.New("POTATO"))
			})
			It("should return the error", func() {
				_, err := reflexClient.GetAddressesViaRouter()
				Expect(err).To(MatchError("POTATO"))
			})
		})
		Context("response cannot be unmashalled from json", func() {
			BeforeEach(func() {
				resp.Body = ioutil.NopCloser(strings.NewReader(`%%%%`))
			})
			It("should return the error", func() {
				_, err := reflexClient.GetAddressesViaRouter()
				Expect(err).To(MatchError(ContainSubstring("invalid character")))
			})
		})
	})

	Describe("CheckInstance", func() {
		It("makes an HTTP request for the app port on the app instance", func() {
			reflexClient.CheckInstance("2.2.2.2")

			Expect(httpClient.GetCallCount()).To(Equal(1))
			Expect(httpClient.GetArgsForCall(0)).To(Equal("http://2.2.2.2:1234/peers"))
		})
		Context("when the response body includes the target address", func() {
			It("returns true", func() {
				Expect(reflexClient.CheckInstance("2.2.2.2")).To(BeTrue())
			})
		})
		Context("when the response body does not include the target address", func() {
			It("returns false", func() {
				Expect(reflexClient.CheckInstance("4.4.4.2")).To(BeFalse())
			})
		})
		Context("when the http client returns an error", func() {
			BeforeEach(func() {
				httpClient.GetReturns(nil, errors.New("POTATO"))
			})
			It("should return false", func() {
				Expect(reflexClient.CheckInstance("2.2.2.2")).To(BeFalse())
			})
		})
	})
})
