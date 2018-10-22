package psclient_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"policy-server/psclient"

	"code.cloudfoundry.org/cf-networking-helpers/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		jsonClient *fakes.JSONClient
		client     *psclient.Client
		token      string
	)

	BeforeEach(func() {
		jsonClient = &fakes.JSONClient{}
		client = &psclient.Client{
			JsonClient: jsonClient,
		}
		token = "some-token"
	})

	Describe("Destinations", func() {
		var (
			destination1, destination2 psclient.Destination
		)

		BeforeEach(func() {
			destination1 = psclient.Destination{
				Name:        "meow-dest",
				Description: "cats rule",
				Protocol:    "tcp",
				IPs: []psclient.IPRange{
					{
						Start: "1.2.3.4",
						End:   "1.2.3.5",
					},
				},
				Ports: []psclient.Port{
					{
						Start: 8080,
						End:   9090,
					},
				},
			}

			destination2 = psclient.Destination{
				Name:        "bark-dest",
				Description: "dogs drool",
				Protocol:    "tcp",
				IPs: []psclient.IPRange{
					{
						Start: "2.2.3.4",
						End:   "2.2.3.5",
					},
				},
				Ports: []psclient.Port{
					{
						Start: 8081,
						End:   9091,
					},
				},
			}
		})

		Describe("create destinations", func() {
			BeforeEach(func() {
				jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					respBytes := []byte(`{
           				"destinations": [ { "id": "some-dest-guid" }, { "id": "some-other-dest-guid" }  ]
                	}`)
					json.Unmarshal(respBytes, respData)
					return nil
				}
			})
			It("creates a destination and returns a guid", func() {
				createdDestinations, err := client.CreateDestinations(token, destination1, destination2)
				Expect(err).NotTo(HaveOccurred())
				expectedDestinations := []psclient.Destination{
					{GUID: "some-dest-guid"},
					{GUID: "some-other-dest-guid"},
				}
				Expect(createdDestinations).To(Equal(expectedDestinations))

				Expect(jsonClient.DoCallCount()).To(Equal(1))
				passedMethod, passedRoute, passedReqData, _, passedToken := jsonClient.DoArgsForCall(0)
				Expect(passedMethod).To(Equal("POST"))
				Expect(passedRoute).To(Equal("/networking/v1/external/destinations"))

				Expect(passedReqData).To(Equal(psclient.DestinationList{
					Destinations: []psclient.Destination{destination1, destination2},
				}))
				Expect(passedToken).To(Equal("Bearer some-token"))
			})

			It("returns an error when the json client do fails", func() {
				jsonClient.DoStub = nil
				jsonClient.DoReturns(errors.New("failed to do"))
				_, err := client.CreateDestinations(token, destination1)
				Expect(err).To(MatchError("json client do: failed to do"))
			})
		})

		Describe("listing destinations", func() {
			BeforeEach(func() {
				jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					respBytes := []byte(`{
           				"destinations": [ { "id": "some-dest-guid" }, { "id": "some-other-dest-guid" }  ]
                	}`)
					json.Unmarshal(respBytes, respData)
					return nil
				}
			})

			It("returns a list of destinations", func() {
				foundDestinations, err := client.ListDestinations(token, psclient.ListDestinationsOptions{})
				Expect(err).NotTo(HaveOccurred())
				expectedDestinations := []psclient.Destination{
					{GUID: "some-dest-guid"},
					{GUID: "some-other-dest-guid"},
				}
				Expect(foundDestinations).To(Equal(expectedDestinations))

				Expect(jsonClient.DoCallCount()).To(Equal(1))
				passedMethod, passedRoute, passedReqData, _, passedToken := jsonClient.DoArgsForCall(0)
				Expect(passedMethod).To(Equal("GET"))
				Expect(passedRoute).To(Equal("/networking/v1/external/destinations?"))

				Expect(passedReqData).To(BeNil())
				Expect(passedToken).To(Equal("Bearer some-token"))
			})

			Context("when using query names", func() {
				It("returns a list of destinations", func() {
					queryParams := psclient.ListDestinationsOptions{
						QueryNames: []string{"some-dest-name", "some-other-dest-name"},
					}
					_, err := client.ListDestinations(token, queryParams)
					Expect(err).NotTo(HaveOccurred())

					Expect(jsonClient.DoCallCount()).To(Equal(1))
					_, passedRoute, _, _, _ := jsonClient.DoArgsForCall(0)
					Expect(passedRoute).To(Equal("/networking/v1/external/destinations?name=some-dest-name%2Csome-other-dest-name"))
				})
			})

			Context("when using query ids", func() {
				It("returns a list of destinations", func() {
					queryParams := psclient.ListDestinationsOptions{
						QueryIDs: []string{"some-dest-guid", "some-other-dest-guid"},
					}
					_, err := client.ListDestinations(token, queryParams)
					Expect(err).NotTo(HaveOccurred())

					Expect(jsonClient.DoCallCount()).To(Equal(1))
					_, passedRoute, _, _, _ := jsonClient.DoArgsForCall(0)
					Expect(passedRoute).To(Equal("/networking/v1/external/destinations?id=some-dest-guid%2Csome-other-dest-guid"))
				})
			})

			It("returns an error when the json client do fails", func() {
				jsonClient.DoStub = nil
				jsonClient.DoReturns(errors.New("failed to do"))
				_, err := client.ListDestinations(token, psclient.ListDestinationsOptions{})
				Expect(err).To(MatchError("json client do: failed to do"))
			})
		})

		Describe("UpdateDestinations", func() {
			var destinationToUpdate1, destinationToUpdate2 psclient.Destination
			BeforeEach(func() {
				jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					respBytes := []byte(`{
						"destinations": [
						{
							 "id": "guid-received-from-server-1",
					     "name": "name-received-from-server",
					     "description": "description-received-from-server",
					     "ips": [{"start": "8.8.8.8", "end": "8.8.8.8"}],
					     "ports": [{"start": 8080, "end": 8080}],
					     "protocol": "tcp"
					  },
						{
							 "id": "guid-received-from-server-2",
					     "name": "name-received-from-server",
					     "description": "description-received-from-server",
					     "ips": [{"start": "8.8.8.8", "end": "8.8.8.8"}],
					     "ports": [{"start": 8080, "end": 8080}],
					     "protocol": "tcp"
					  }
						]
					}`)
					Expect(json.Unmarshal(respBytes, respData)).To(Succeed())
					return nil
				}

				destinationToUpdate1 = destination1
				destinationToUpdate1.GUID = "guid-of-dest-to-update-1"
				destinationToUpdate2 = destination2
				destinationToUpdate2.GUID = "guid-of-dest-to-update-2"
			})

			It("updates the destination", func() {
				updatedDestinations, err := client.UpdateDestinations(token, destinationToUpdate1, destinationToUpdate2)
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedDestinations).To(Equal([]psclient.Destination{
					{
						GUID:        "guid-received-from-server-1",
						Name:        "name-received-from-server",
						Description: "description-received-from-server",
						IPs:         []psclient.IPRange{{Start: "8.8.8.8", End: "8.8.8.8"}},
						Ports:       []psclient.Port{{Start: 8080, End: 8080}},
						Protocol:    "tcp",
					},
					{
						GUID:        "guid-received-from-server-2",
						Name:        "name-received-from-server",
						Description: "description-received-from-server",
						IPs:         []psclient.IPRange{{Start: "8.8.8.8", End: "8.8.8.8"}},
						Ports:       []psclient.Port{{Start: 8080, End: 8080}},
						Protocol:    "tcp",
					},
				}))

				Expect(jsonClient.DoCallCount()).To(Equal(1))
				passedMethod, passedRoute, passedReqData, _, passedToken := jsonClient.DoArgsForCall(0)
				Expect(passedMethod).To(Equal("PUT"))
				Expect(passedRoute).To(Equal("/networking/v1/external/destinations"))

				Expect(passedReqData).To(Equal(psclient.DestinationList{
					Destinations: []psclient.Destination{
						destinationToUpdate1,
						destinationToUpdate2,
					},
				}))
				Expect(passedToken).To(Equal("Bearer some-token"))
			})

			Context("when no destinations are passed", func() {
				It("returns a helpful error", func() {
					_, err := client.UpdateDestinations(token)
					Expect(err).To(MatchError("destinations to be updated must not be empty"))
				})
			})

			Context("when the caller forgets to set the GUID field on the Destination", func() {
				BeforeEach(func() {
					destinationToUpdate1.GUID = ""
				})
				It("returns early with a helpful error", func() {
					_, err := client.UpdateDestinations(token, destinationToUpdate1)
					Expect(err).To(MatchError("destinations to be updated must have an ID"))
				})
			})

			Context("when the json client do fails", func() {
				It("wraps and returns the error", func() {
					jsonClient.DoStub = nil
					jsonClient.DoReturns(errors.New("failed to do"))
					_, err := client.UpdateDestinations(token, destinationToUpdate1)
					Expect(err).To(MatchError("json client do: failed to do"))
				})
			})
		})

		Describe("DeleteDestination", func() {
			var (
				destinationToDelete psclient.Destination
				destinationResp     psclient.DestinationList
			)
			BeforeEach(func() {
				destinationResp = psclient.DestinationList{
					Destinations: []psclient.Destination{{
						GUID:     "response-dest-guid",
						Protocol: "tcp",
						IPs: []psclient.IPRange{
							{Start: "1.1.1.1", End: "1.1.1.5"},
						},
						Ports: []psclient.Port{
							{Start: 1234, End: 2345},
						},
						Name:        "destinationObject",
						Description: "description",
					}},
				}
				destinationRespBytes, err := json.Marshal(destinationResp)
				Expect(err).NotTo(HaveOccurred())

				jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					json.Unmarshal(destinationRespBytes, respData)
					return nil
				}

				destinationToDelete = psclient.Destination{GUID: "dest-guid"}
			})

			It("deletes destination", func() {
				deletedDestination, err := client.DeleteDestination(token, destinationToDelete)
				Expect(err).NotTo(HaveOccurred())

				Expect(deletedDestination).To(Equal(destinationResp.Destinations[0]))

				Expect(jsonClient.DoCallCount()).To(Equal(1))
				passedMethod, passedRoute, _, _, passedToken := jsonClient.DoArgsForCall(0)
				Expect(passedMethod).To(Equal("DELETE"))
				Expect(passedRoute).To(Equal("/networking/v1/external/destinations/dest-guid"))
				Expect(passedToken).To(Equal("Bearer some-token"))
			})

			It("returns an error when the json client do fails", func() {
				jsonClient.DoStub = nil
				jsonClient.DoReturns(errors.New("failed to do"))
				_, err := client.DeleteDestination(token, destination1)
				Expect(err).To(MatchError("json client do: failed to do"))
			})
		})
	})

	Describe("CreateEgressPolicy", func() {
		var (
			egressPolicy psclient.EgressPolicy
		)

		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{
                    "egress_policies": [ { "id": "some-egress-policy-guid" } ]
                }`)
				json.Unmarshal(respBytes, respData)
				return nil
			}

			egressPolicy = psclient.EgressPolicy{
				Source: psclient.EgressPolicySource{
					Type: "app",
					ID:   "some-app-guid",
				},
				Destination: psclient.Destination{
					GUID: "some-dest-guid",
				},
			}
		})

		It("creates an egress policy and returns a guid", func() {
			guid, err := client.CreateEgressPolicy(egressPolicy, token)
			Expect(err).NotTo(HaveOccurred())
			Expect(guid).To(Equal("some-egress-policy-guid"))

			Expect(jsonClient.DoCallCount()).To(Equal(1))
			passedMethod, passedRoute, passedReqData, _, passedToken := jsonClient.DoArgsForCall(0)
			Expect(passedMethod).To(Equal("POST"))
			Expect(passedRoute).To(Equal("/networking/v1/external/egress_policies"))

			Expect(passedReqData).To(Equal(psclient.EgressPolicyList{
				EgressPolicies: []psclient.EgressPolicy{egressPolicy}},
			))
			Expect(passedToken).To(Equal("Bearer some-token"))
		})

		It("returns an error when the json client do fails", func() {
			jsonClient.DoStub = nil
			jsonClient.DoReturns(errors.New("failed to do"))
			_, err := client.CreateEgressPolicy(egressPolicy, token)
			Expect(err).To(MatchError("json client do: failed to do"))
		})
	})

	Describe("DeleteEgressPolicy", func() {
		var (
			expectedEgressPolicy psclient.EgressPolicy
			egressPolicyGUID     string
		)

		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{
					"egress_policies": [
						{
							"id": "some-egress-policy-guid",
							"source": {
								"type": "app",
								"id":   "some-app-guid"
							},
							"destination": {
								"id": "some-dest-guid"
							}
						}
					]
				}`)
				err := json.Unmarshal(respBytes, respData)
				Expect(err).NotTo(HaveOccurred())
				return nil
			}

			expectedEgressPolicy = psclient.EgressPolicy{
				GUID: "some-egress-policy-guid",
				Source: psclient.EgressPolicySource{
					Type: "app",
					ID:   "some-app-guid",
				},
				Destination: psclient.Destination{
					GUID: "some-dest-guid",
				},
			}
			egressPolicyGUID = "some-egress-policy-guid"
		})

		It("deletes an egress policy provided a guid and returns the deleted egress policy", func() {
			egressPolicy, err := client.DeleteEgressPolicy(egressPolicyGUID, token)
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicy).To(Equal(expectedEgressPolicy))

			Expect(jsonClient.DoCallCount()).To(Equal(1))
			passedMethod, passedRoute, passedReqData, _, passedToken := jsonClient.DoArgsForCall(0)
			Expect(passedMethod).To(Equal("DELETE"))
			Expect(passedRoute).To(Equal(fmt.Sprintf("/networking/v1/external/egress_policies/%s", egressPolicyGUID)))
			Expect(passedReqData).To(BeEmpty())
			Expect(passedToken).To(Equal("Bearer some-token"))
		})

		It("returns an error when the json client do fails", func() {
			jsonClient.DoStub = nil
			jsonClient.DoReturns(errors.New("failed to do"))
			_, err := client.DeleteEgressPolicy(egressPolicyGUID, token)
			Expect(err).To(MatchError("json client do: failed to do"))
		})
	})

	Describe("ListEgressPolicy", func() {
		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				respBytes := []byte(`{
                    "total_egress_policies": 1,
                    "egress_policies": [
                        {
                            "id": "some-egress-policy-guid",
                            "source": {
                                "type": "app",
                                "id": "some-app-guid"
                            },
                            "destination": {
                                "id": "some-dest-guid"
                            }
                        }
                    ]
                }`)
				json.Unmarshal(respBytes, respData)
				return nil
			}
		})

		It("lists all egress policies", func() {
			policyList, err := client.ListEgressPolicies(token)
			Expect(err).NotTo(HaveOccurred())
			Expect(policyList.TotalEgressPolicies).To(Equal(1))
			Expect(policyList.EgressPolicies).To(ConsistOf(psclient.EgressPolicy{
				GUID: "some-egress-policy-guid",
				Source: psclient.EgressPolicySource{
					Type: "app",
					ID:   "some-app-guid",
				},
				Destination: psclient.Destination{
					GUID: "some-dest-guid",
				},
			}))

			Expect(jsonClient.DoCallCount()).To(Equal(1))
			passedMethod, passedRoute, passedReqData, _, passedToken := jsonClient.DoArgsForCall(0)
			Expect(passedMethod).To(Equal("GET"))
			Expect(passedRoute).To(Equal("/networking/v1/external/egress_policies"))
			Expect(passedReqData).To(BeEmpty())
			Expect(passedToken).To(Equal("Bearer some-token"))
		})

		It("returns an error when the json client do fails", func() {
			jsonClient.DoStub = nil
			jsonClient.DoReturns(errors.New("failed to do"))
			_, err := client.ListEgressPolicies(token)
			Expect(err).To(MatchError("list egress policies api call: failed to do"))
		})
	})
})
