package policy_client_test

import (
	"encoding/json"
	"errors"
	"lib/fakes"
	"lib/policy_client"
	"net/http"
	"policy-server/api/api_v0"

	hfakes "code.cloudfoundry.org/cf-networking-helpers/fakes"

	"code.cloudfoundry.org/cf-networking-helpers/json_client"

	"policy-server/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/ginkgo/extensions/table"
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
		fakeChunker.ChunkReturns([][]api_v0.Policy{
			{
				{
					Source: api_v0.Source{
						ID: "some-app-guid",
					},
					Destination: api_v0.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
				{
					Source: api_v0.Source{
						ID: "some-app-guid-2",
					},
					Destination: api_v0.Destination{
						ID:       "some-other-app-guid-2",
						Port:     8091,
						Protocol: "tcp",
					},
				},
			},
			{
				{
					Source: api_v0.Source{
						ID: "some-app-guid-3",
					},
					Destination: api_v0.Destination{
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

	Describe("GetAPIVersion", func() {
		DescribeTable("when getting version succeeds", func(ccRootEndpointJson string, expectedVersion int) {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(ccRootEndpointJson)
				json.Unmarshal(respBytes, respData)
				return nil
			}
			version, err := client.GetAPIVersion()
			Expect(err).NotTo(HaveOccurred())

			Expect(jsonClient.DoCallCount()).To(Equal(1))
			method, route, reqData, _, token := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/"))
			Expect(reqData).To(BeNil())
			Expect(token).To(BeEmpty())

			Expect(version).To(Equal(expectedVersion))
		},
			Entry("when key is 'network_policy' (old capi)", `{
			   "links": {
				  "network_policy": {
					 "href": "https://api.bosh-lite.com/networking/v0/external"
				  }
			   }
			}
			`, 0),
			Entry("when no networking key is present at all (old old capi)",
				`{"links": { }}`, -1),
			Entry("when keys are 'network_policy_v0' and 'network_policy_v1' (newer capi)", `{
			   "links": {
				  "network_policy_v0": {
					 "href": "https://api.bosh-lite.com/networking/v0/external"
				  },
				  "network_policy_v1": {
					 "href": "https://api.bosh-lite.com/networking/v1/external"
				  }
			   }
			}
			`, 1),
		)

		Context("when the json client fails", func() {
			BeforeEach(func() {
				jsonClient.DoReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := client.GetAPIVersion()
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
				_, err := client.GetAPIVersion()
				Expect(err).To(MatchError("418 I'm a teapot: some-error"))
			})
		})
	})

	Describe("GetPolicies", func() {
		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "ports": { "start": 8090, "end": 8100 } } } ] }`)
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
			Expect(route).To(Equal("/networking/v1/external/policies"))
			Expect(reqData).To(BeNil())

			Expect(policies).To(Equal([]api.Policy{
				{
					Source: api.Source{
						ID: "some-app-guid",
					},
					Destination: api.Destination{
						ID: "some-other-app-guid",
						Ports: api.Ports{
							Start: 8090,
							End:   8100,
						},
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
				respBytes := []byte(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "ports": { "start": 8090, "end": 8100 } } } ] }`)
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
			Expect(route).To(Equal("/networking/v1/external/policies?id=some-app-guid,another-app-guid"))
			Expect(reqData).To(BeNil())
			Expect(policies).To(Equal([]api.Policy{
				{
					Source: api.Source{
						ID: "some-app-guid",
					},
					Destination: api.Destination{
						ID: "some-other-app-guid",
						Ports: api.Ports{
							Start: 8090,
							End:   8100,
						},
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

	Describe("GetPoliciesV0", func() {
		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`)
				json.Unmarshal(respBytes, respData)
				return nil
			}
		})
		It("does the right json http client request", func() {
			policies, err := client.GetPoliciesV0("some-token")
			Expect(err).NotTo(HaveOccurred())

			Expect(jsonClient.DoCallCount()).To(Equal(1))
			method, route, reqData, _, token := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/networking/v0/external/policies"))
			Expect(reqData).To(BeNil())

			Expect(policies).To(Equal([]api_v0.Policy{
				{
					Source: api_v0.Source{
						ID: "some-app-guid",
					},
					Destination: api_v0.Destination{
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
				_, err := client.GetPoliciesV0("some-token")
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
				_, err := client.GetPoliciesV0("some-token")
				Expect(err).To(MatchError("418 I'm a teapot: some-error"))
			})
		})
	})

	Describe("GetPoliciesV0ByID", func() {
		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`)
				json.Unmarshal(respBytes, respData)
				return nil
			}
		})
		It("does the right json http client request", func() {
			policies, err := client.GetPoliciesV0ByID("some-token", "some-app-guid", "another-app-guid")
			Expect(err).NotTo(HaveOccurred())

			Expect(jsonClient.DoCallCount()).To(Equal(1))
			method, route, reqData, _, token := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/networking/v0/external/policies?id=some-app-guid,another-app-guid"))
			Expect(reqData).To(BeNil())
			Expect(policies).To(Equal([]api_v0.Policy{
				{
					Source: api_v0.Source{
						ID: "some-app-guid",
					},
					Destination: api_v0.Destination{
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
				_, err := client.GetPoliciesV0ByID("some-token", "some-id")
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
				_, err := client.GetPoliciesV0ByID("some-token", "some-id")
				Expect(err).To(MatchError("418 I'm a teapot: some-error"))
			})
		})
	})

	Describe("AddPoliciesV0", func() {
		var policiesToAdd []api_v0.Policy
		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{}`)
				json.Unmarshal(respBytes, respData)
				return nil
			}

			policiesToAdd = []api_v0.Policy{
				{
					Source: api_v0.Source{
						ID: "some-app-guid",
					},
					Destination: api_v0.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
			}
		})
		It("does the right json http client request and passes the authorization token", func() {
			err := client.AddPoliciesV0("some-token", policiesToAdd)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeChunker.ChunkCallCount()).To(Equal(1))
			Expect(fakeChunker.ChunkArgsForCall(0)).To(Equal(policiesToAdd))

			Expect(jsonClient.DoCallCount()).To(Equal(2))
			method, route, reqData, _, token := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("POST"))
			Expect(route).To(Equal("/networking/v0/external/policies"))
			Expect(reqData).To(Equal(map[string][]api_v0.Policy{
				"policies": []api_v0.Policy{{
					Source: api_v0.Source{
						ID: "some-app-guid",
					},
					Destination: api_v0.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
					{
						Source: api_v0.Source{
							ID: "some-app-guid-2",
						},
						Destination: api_v0.Destination{
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
			Expect(reqData).To(Equal(map[string][]api_v0.Policy{
				"policies": []api_v0.Policy{
					{
						Source: api_v0.Source{
							ID: "some-app-guid-3",
						},
						Destination: api_v0.Destination{
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
				err := client.AddPoliciesV0("some-token", policiesToAdd)
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
				err := client.AddPoliciesV0("some-token", policiesToAdd)
				Expect(err).To(MatchError("418 I'm a teapot: some-error"))
			})
		})
	})

	Describe("AddPolicies", func() {
		var policiesToAdd []api.Policy
		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{}`)
				json.Unmarshal(respBytes, respData)
				return nil
			}

			policiesToAdd = []api.Policy{{
				Source: api.Source{
					ID: "some-app-guid",
				},
				Destination: api.Destination{
					ID:       "some-other-app-guid",
					Ports:    api.Ports{Start: 8080, End: 8090},
					Protocol: "tcp",
				},
			},
				{
					Source: api.Source{
						ID: "some-app-guid-2",
					},
					Destination: api.Destination{
						ID:       "some-other-app-guid-2",
						Ports:    api.Ports{Start: 8091, End: 8100},
						Protocol: "tcp",
					},
				},
			}
		})
		It("does the right json http client request and passes the authorization token", func() {
			err := client.AddPolicies("some-token", policiesToAdd)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeChunker.ChunkCallCount()).To(Equal(0))

			Expect(jsonClient.DoCallCount()).To(Equal(1))
			method, route, reqData, _, token := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("POST"))
			Expect(route).To(Equal("/networking/v1/external/policies"))
			Expect(reqData).To(Equal(map[string][]api.Policy{
				"policies": []api.Policy{{
					Source: api.Source{
						ID: "some-app-guid",
					},
					Destination: api.Destination{
						ID:       "some-other-app-guid",
						Ports:    api.Ports{Start: 8080, End: 8090},
						Protocol: "tcp",
					},
				},
					{
						Source: api.Source{
							ID: "some-app-guid-2",
						},
						Destination: api.Destination{
							ID:       "some-other-app-guid-2",
							Ports:    api.Ports{Start: 8091, End: 8100},
							Protocol: "tcp",
						},
					},
				}},
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

	Describe("DeletePoliciesV0", func() {
		var policiesToDelete []api_v0.Policy

		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{}`)
				json.Unmarshal(respBytes, respData)
				return nil
			}

			policiesToDelete = []api_v0.Policy{
				{
					Source: api_v0.Source{
						ID: "some-app-guid",
					},
					Destination: api_v0.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
			}
		})
		It("does the right json http client request", func() {
			err := client.DeletePoliciesV0("some-token", policiesToDelete)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeChunker.ChunkCallCount()).To(Equal(1))
			Expect(fakeChunker.ChunkArgsForCall(0)).To(Equal(policiesToDelete))

			Expect(jsonClient.DoCallCount()).To(Equal(2))
			method, route, reqData, _, token := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("POST"))
			Expect(route).To(Equal("/networking/v0/external/policies/delete"))
			Expect(reqData).To(Equal(map[string][]api_v0.Policy{
				"policies": []api_v0.Policy{{
					Source: api_v0.Source{
						ID: "some-app-guid",
					},
					Destination: api_v0.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
					{
						Source: api_v0.Source{
							ID: "some-app-guid-2",
						},
						Destination: api_v0.Destination{
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
			Expect(reqData).To(Equal(map[string][]api_v0.Policy{
				"policies": []api_v0.Policy{
					{
						Source: api_v0.Source{
							ID: "some-app-guid-3",
						},
						Destination: api_v0.Destination{
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
				err := client.DeletePoliciesV0("some-token", policiesToDelete)
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
				err := client.DeletePoliciesV0("some-token", policiesToDelete)
				Expect(err).To(MatchError("418 I'm a teapot: some-error"))
			})
		})
	})

	Describe("DeletePolicies", func() {
		var policiesToDelete []api.Policy

		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{}`)
				json.Unmarshal(respBytes, respData)
				return nil
			}

			policiesToDelete = []api.Policy{
				{
					Source: api.Source{
						ID: "some-app-guid",
					},
					Destination: api.Destination{
						ID: "some-other-app-guid",
						Ports: api.Ports{
							Start: 1234,
							End:   2345,
						},
						Protocol: "tcp",
					},
				},
			}
		})
		It("does the right json http client request", func() {
			err := client.DeletePolicies("some-token", policiesToDelete)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeChunker.ChunkCallCount()).To(Equal(0))

			Expect(jsonClient.DoCallCount()).To(Equal(1))
			method, route, reqData, _, token := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("POST"))
			Expect(route).To(Equal("/networking/v1/external/policies/delete"))
			Expect(reqData).To(Equal(map[string][]api.Policy{
				"policies": []api.Policy{
					{
						Source: api.Source{
							ID: "some-app-guid",
						},
						Destination: api.Destination{
							ID: "some-other-app-guid",
							Ports: api.Ports{
								Start: 1234,
								End:   2345,
							},
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
