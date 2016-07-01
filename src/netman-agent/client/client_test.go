package client_test

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"netman-agent/client"
	"netman-agent/fakes"
	"strings"

	lfakes "lib/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		netman_client    *client.Client
		httpClient       *fakes.HTTPClient
		fakeMarshaler    *lfakes.Marshaler
		returnedResponse *http.Response
	)

	BeforeEach(func() {
		httpClient = &fakes.HTTPClient{}
		fakeMarshaler = &lfakes.Marshaler{}
		fakeMarshaler.MarshalStub = json.Marshal
		netman_client = client.New(httpClient, "http://some.url", fakeMarshaler)

		returnedResponse = &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`{}`)),
		}
		httpClient.DoReturns(returnedResponse, nil)
	})

	Describe("Add", func() {
		It("posts the container data to the /cni_result endpoint", func() {
			err := netman_client.Add("some-container-id", "some-group-id", net.ParseIP("1.2.3.4"))
			Expect(err).NotTo(HaveOccurred())

			Expect(httpClient.DoCallCount()).To(Equal(1))
			request := httpClient.DoArgsForCall(0)
			body, err := ioutil.ReadAll(request.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(request.Method).To(Equal("POST"))
			Expect(body).To(MatchJSON(`{
				"container_id": "some-container-id",
				"group_id": "some-group-id",
				"ip" : "1.2.3.4"}`))
		})

		Context("when marshaling the body fails", func() {
			BeforeEach(func() {
				fakeMarshaler.MarshalStub = func(interface{}) ([]byte, error) {
					return nil, errors.New("banana")
				}
			})
			It("returns the error", func() {
				err := netman_client.Add("some-container-id", "some-group-id", net.ParseIP("1.2.3.4"))
				Expect(err).To(MatchError("json marshal: banana"))
			})
		})

		Context("when the request cannot be constructed", func() {
			BeforeEach(func() {
				netman_client = client.New(httpClient, "%%%", fakeMarshaler)
			})
			It("return the error", func() {
				err := netman_client.Add("some-container-id", "some-group-id", net.ParseIP("1.2.3.4"))
				Expect(err).To(MatchError(ContainSubstring("constructing request: parse")))
			})
		})

		Context("when the http client fails", func() {
			BeforeEach(func() {
				httpClient.DoReturns(nil, errors.New("potato"))
			})
			It("returns the error", func() {
				err := netman_client.Add("some-container-id", "some-group-id", net.ParseIP("1.2.3.4"))
				Expect(err).To(MatchError("http client do: potato"))
			})
		})

		Context("when netman responds with an unexpected status code", func() {
			BeforeEach(func() {
				returnedResponse = &http.Response{
					StatusCode: 420,
					Body:       ioutil.NopCloser(strings.NewReader(`{}`)),
				}
				httpClient.DoReturns(returnedResponse, nil)
			})

			It("returns a useful error", func() {
				err := netman_client.Add("some-container-id", "some-group-id", net.ParseIP("1.2.3.4"))
				Expect(err).To(MatchError("unexpected status code 420"))
			})
		})
	})

	Describe("Del", func() {
		It("makes a delete request to the /cni_result endpoint", func() {
			err := netman_client.Del("some-container-id")
			Expect(err).NotTo(HaveOccurred())

			Expect(httpClient.DoCallCount()).To(Equal(1))
			request := httpClient.DoArgsForCall(0)
			body, err := ioutil.ReadAll(request.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(request.Method).To(Equal("DELETE"))
			Expect(body).To(MatchJSON(`{
				"container_id": "some-container-id"
				}`))
		})
	})
})
