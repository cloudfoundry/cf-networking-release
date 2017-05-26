package policy_client_test

import (
	"encoding/json"
	"errors"
	"lib/models"
	"lib/policy_client"

	hfakes "code.cloudfoundry.org/cf-networking-helpers/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("InternalClient", func() {
	var (
		client     *policy_client.InternalClient
		jsonClient *hfakes.JSONClient
	)

	BeforeEach(func() {
		jsonClient = &hfakes.JSONClient{}
		client = &policy_client.InternalClient{
			JsonClient: jsonClient,
		}
	})

	Describe("GetPolicies", func() {
		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{ "policies": [ {"source": { "id": "some-app-guid", "tag": "BEEF" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`)
				json.Unmarshal(respBytes, respData)
				return nil
			}
		})
		It("does the right json http client request", func() {
			policies, err := client.GetPolicies()
			Expect(err).NotTo(HaveOccurred())

			Expect(jsonClient.DoCallCount()).To(Equal(1))
			method, route, reqData, _, token := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/networking/v0/internal/policies"))
			Expect(reqData).To(BeNil())

			Expect(policies).To(Equal([]models.Policy{
				{
					Source: models.Source{
						ID:  "some-app-guid",
						Tag: "BEEF",
					},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
			},
			))
			Expect(token).To(BeEmpty())
		})

		Context("when the json client fails", func() {
			BeforeEach(func() {
				jsonClient.DoReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := client.GetPolicies()
				Expect(err).To(MatchError("banana"))
			})
		})
	})

	Describe("GetPoliciesByID", func() {
		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{ "policies": [ {"source": { "id": "some-app-guid", "tag": "BEEF" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`)
				json.Unmarshal(respBytes, respData)
				return nil
			}
		})
		It("does the right json http client request", func() {
			policies, err := client.GetPoliciesByID("some-app-guid", "some-other-app-guid")
			Expect(err).NotTo(HaveOccurred())

			Expect(jsonClient.DoCallCount()).To(Equal(1))
			method, route, reqData, _, token := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/networking/v0/internal/policies?id=some-app-guid,some-other-app-guid"))
			Expect(reqData).To(BeNil())

			Expect(policies).To(Equal([]models.Policy{
				{
					Source: models.Source{
						ID:  "some-app-guid",
						Tag: "BEEF",
					},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
			},
			))
			Expect(token).To(BeEmpty())
		})

		Context("when the json client fails", func() {
			BeforeEach(func() {
				jsonClient.DoReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := client.GetPoliciesByID("foo")
				Expect(err).To(MatchError("banana"))
			})
		})

		Context("when ids is empty", func() {
			BeforeEach(func() {})
			It("returns an error and does not call the json http client", func() {
				policies, err := client.GetPoliciesByID()
				Expect(err).To(MatchError("ids cannot be empty"))
				Expect(policies).To(BeNil())
				Expect(jsonClient.DoCallCount()).To(Equal(0))
			})
		})
	})

	Describe("HealthCheck", func() {
		BeforeEach(func(){
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{ "healthcheck": true }`)
				json.Unmarshal(respBytes, respData)
				return nil
			}
		})

		It("Returns if the server is up", func(){
			health, err := client.HealthCheck()
			Expect(err).NotTo(HaveOccurred())
			Expect(health).To(Equal(true))

			Expect(jsonClient.DoCallCount()).To(Equal(1))
			method, route, reqData, _, token := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/networking/v0/internal/healthcheck"))
			Expect(reqData).To(BeNil())
			Expect(token).To(BeEmpty())
		})

		Context("when the json client fails", func() {
			BeforeEach(func() {
				jsonClient.DoReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := client.HealthCheck()
				Expect(err).To(MatchError("banana"))
			})
		})	
	})

})
