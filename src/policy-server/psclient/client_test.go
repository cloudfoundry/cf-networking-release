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

		Describe("create", func() {
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
				Destination: psclient.EgressPolicyDestination{
					ID: "some-dest-guid",
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
				Destination: psclient.EgressPolicyDestination{
					ID: "some-dest-guid",
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
				Destination: psclient.EgressPolicyDestination{
					ID: "some-dest-guid",
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
