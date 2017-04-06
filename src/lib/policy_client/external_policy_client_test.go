package policy_client_test

import (
	"encoding/json"
	"errors"
	"lib/fakes"
	"lib/models"
	"lib/policy_client"
	"net/http"

	hfakes "code.cloudfoundry.org/go-db-helpers/fakes"

	"code.cloudfoundry.org/go-db-helpers/json_client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ExternalClient", func() {
	var (
		client      *policy_client.ExternalClient
		fakeChunker *fakes.Chunker
		jsonClient  *hfakes.JSONClient
	)

	BeforeEach(func() {
		jsonClient = &hfakes.JSONClient{}
		fakeChunker = &fakes.Chunker{}
		fakeChunker.ChunkReturns([][]models.Policy{
			[]models.Policy{
				{
					Source: models.Source{
						ID: "some-app-guid",
					},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
				{
					Source: models.Source{
						ID: "some-app-guid-2",
					},
					Destination: models.Destination{
						ID:       "some-other-app-guid-2",
						Port:     8091,
						Protocol: "tcp",
					},
				},
			},
			[]models.Policy{
				{
					Source: models.Source{
						ID: "some-app-guid-3",
					},
					Destination: models.Destination{
						ID:       "some-other-app-guid-3",
						Port:     8092,
						Protocol: "tcp",
					},
				},
			},
		})
		client = &policy_client.ExternalClient{
			JsonClient: jsonClient,
			Chunker:    fakeChunker,
		}
	})

	Describe("GetPolicies", func() {
		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`)
				json.Unmarshal(respBytes, respData)
				return nil
			}
		})
		It("does the right json http client request", func() {
			policies, err := client.GetPolicies("some-token")
			Expect(err).NotTo(HaveOccurred())

			Expect(jsonClient.DoCallCount()).To(Equal(1))
			method, route, reqData, _, token := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/networking/v0/external/policies"))
			Expect(reqData).To(BeNil())

			Expect(policies).To(Equal([]models.Policy{
				{
					Source: models.Source{
						ID: "some-app-guid",
					},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
			},
			))
			Expect(token).To(Equal("some-token"))
		})
		Context("when the json client fails", func() {
			BeforeEach(func() {
				jsonClient.DoReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := client.GetPolicies("some-token")
				Expect(err).To(MatchError("banana"))
			})
		})
		Context("when the json client gets a bad status code", func() {
			BeforeEach(func() {
				jsonClient.DoReturns(&json_client.HttpResponseCodeError{
					StatusCode: http.StatusTeapot,
					Message:    "some-error",
				})
			})
			It("parses out the error body", func() {
				_, err := client.GetPolicies("some-token")
				Expect(err).To(MatchError("418 I'm a teapot: some-error"))
			})
		})
	})

	Describe("GetPoliciesByID", func() {
		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`)
				json.Unmarshal(respBytes, respData)
				return nil
			}
		})
		It("does the right json http client request", func() {
			policies, err := client.GetPoliciesByID("some-token", "some-app-guid", "another-app-guid")
			Expect(err).NotTo(HaveOccurred())

			Expect(jsonClient.DoCallCount()).To(Equal(1))
			method, route, reqData, _, token := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/networking/v0/external/policies?id=some-app-guid,another-app-guid"))
			Expect(reqData).To(BeNil())
			Expect(policies).To(Equal([]models.Policy{
				{
					Source: models.Source{
						ID: "some-app-guid",
					},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
			},
			))
			Expect(token).To(Equal("some-token"))
		})
		Context("when the json client fails", func() {
			BeforeEach(func() {
				jsonClient.DoReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := client.GetPoliciesByID("some-token", "some-id")
				Expect(err).To(MatchError("banana"))
			})
		})
		Context("when the json client gets a bad status code", func() {
			BeforeEach(func() {
				jsonClient.DoReturns(&json_client.HttpResponseCodeError{
					StatusCode: http.StatusTeapot,
					Message:    "some-error",
				})
			})
			It("parses out the error body", func() {
				_, err := client.GetPoliciesByID("some-token", "some-id")
				Expect(err).To(MatchError("418 I'm a teapot: some-error"))
			})
		})
	})

	Describe("AddPolicies", func() {
		var policiesToAdd []models.Policy
		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{}`)
				json.Unmarshal(respBytes, respData)
				return nil
			}

			policiesToAdd = []models.Policy{
				{
					Source: models.Source{
						ID: "some-app-guid",
					},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
			}
		})
		It("does the right json http client request and passes the authorization token", func() {
			err := client.AddPolicies("some-token", policiesToAdd)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeChunker.ChunkCallCount()).To(Equal(1))
			Expect(fakeChunker.ChunkArgsForCall(0)).To(Equal(policiesToAdd))

			Expect(jsonClient.DoCallCount()).To(Equal(2))
			method, route, reqData, _, token := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("POST"))
			Expect(route).To(Equal("/networking/v0/external/policies"))
			Expect(reqData).To(Equal(map[string][]models.Policy{
				"policies": []models.Policy{{
					Source: models.Source{
						ID: "some-app-guid",
					},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
					{
						Source: models.Source{
							ID: "some-app-guid-2",
						},
						Destination: models.Destination{
							ID:       "some-other-app-guid-2",
							Port:     8091,
							Protocol: "tcp",
						},
					},
				}},
			))
			Expect(token).To(Equal("some-token"))

			method, route, reqData, _, token = jsonClient.DoArgsForCall(1)
			Expect(method).To(Equal("POST"))
			Expect(route).To(Equal("/networking/v0/external/policies"))
			Expect(reqData).To(Equal(map[string][]models.Policy{
				"policies": []models.Policy{
					{
						Source: models.Source{
							ID: "some-app-guid-3",
						},
						Destination: models.Destination{
							ID:       "some-other-app-guid-3",
							Port:     8092,
							Protocol: "tcp",
						},
					},
				},
			},
			))
			Expect(token).To(Equal("some-token"))
		})
		Context("when the json client fails", func() {
			BeforeEach(func() {
				jsonClient.DoReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				err := client.AddPolicies("some-token", policiesToAdd)
				Expect(err).To(MatchError("banana"))
			})
		})
		Context("when the json client gets a bad status code", func() {
			BeforeEach(func() {
				jsonClient.DoReturns(&json_client.HttpResponseCodeError{
					StatusCode: http.StatusTeapot,
					Message:    "some-error",
				})
			})
			It("parses out the error body", func() {
				err := client.AddPolicies("some-token", policiesToAdd)
				Expect(err).To(MatchError("418 I'm a teapot: some-error"))
			})
		})
	})

	Describe("DeletePolicies", func() {
		var policiesToDelete []models.Policy

		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{}`)
				json.Unmarshal(respBytes, respData)
				return nil
			}

			policiesToDelete = []models.Policy{
				{
					Source: models.Source{
						ID: "some-app-guid",
					},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
			}
		})
		It("does the right json http client request", func() {
			err := client.DeletePolicies("some-token", policiesToDelete)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeChunker.ChunkCallCount()).To(Equal(1))
			Expect(fakeChunker.ChunkArgsForCall(0)).To(Equal(policiesToDelete))

			Expect(jsonClient.DoCallCount()).To(Equal(2))
			method, route, reqData, _, token := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("POST"))
			Expect(route).To(Equal("/networking/v0/external/policies/delete"))
			Expect(reqData).To(Equal(map[string][]models.Policy{
				"policies": []models.Policy{{
					Source: models.Source{
						ID: "some-app-guid",
					},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
					{
						Source: models.Source{
							ID: "some-app-guid-2",
						},
						Destination: models.Destination{
							ID:       "some-other-app-guid-2",
							Port:     8091,
							Protocol: "tcp",
						},
					},
				}},
			))
			Expect(token).To(Equal("some-token"))

			method, route, reqData, _, token = jsonClient.DoArgsForCall(1)
			Expect(method).To(Equal("POST"))
			Expect(route).To(Equal("/networking/v0/external/policies/delete"))
			Expect(reqData).To(Equal(map[string][]models.Policy{
				"policies": []models.Policy{
					{
						Source: models.Source{
							ID: "some-app-guid-3",
						},
						Destination: models.Destination{
							ID:       "some-other-app-guid-3",
							Port:     8092,
							Protocol: "tcp",
						},
					},
				},
			},
			))
			Expect(token).To(Equal("some-token"))
		})
		Context("when the json client fails", func() {
			BeforeEach(func() {
				jsonClient.DoReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				err := client.DeletePolicies("some-token", policiesToDelete)
				Expect(err).To(MatchError("banana"))
			})
		})
		Context("when the json client gets a bad status code", func() {
			BeforeEach(func() {
				jsonClient.DoReturns(&json_client.HttpResponseCodeError{
					StatusCode: http.StatusTeapot,
					Message:    "some-error",
				})
			})
			It("parses out the error body", func() {
				err := client.DeletePolicies("some-token", policiesToDelete)
				Expect(err).To(MatchError("418 I'm a teapot: some-error"))
			})
		})
	})
})
