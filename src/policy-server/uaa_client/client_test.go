package uaa_client_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server"
	"policy-server/fakes"
	"policy-server/uaa_client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		client        *uaa_client.Client
		mockUAAServer *httptest.Server
		handler       http.HandlerFunc
		httpClient    *fakes.HTTPClient
	)

	Describe("GetName", func() {
		BeforeEach(func() {
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/check_token" && r.Method == "POST" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"scope":["network.admin"], "user_name":"some-user"}`))
					return
				}
			})
			mockUAAServer = httptest.NewServer(handler)
			client = &uaa_client.Client{
				Host:       mockUAAServer.URL,
				Name:       "test",
				Secret:     "test",
				HTTPClient: http.DefaultClient,
			}
			httpClient = &fakes.HTTPClient{}
		})

		It("Gets the username by posting to check_token uaa endpoint", func() {
			userName, err := client.GetName("valid-token")
			Expect(err).NotTo(HaveOccurred())

			Expect(userName).To(Equal("some-user"))
		})

		Context("when the response body is not valid json", func() {
			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`%%%%%%%%`))
				})
				mockUAAServer = httptest.NewServer(handler)
				client.Host = mockUAAServer.URL
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
