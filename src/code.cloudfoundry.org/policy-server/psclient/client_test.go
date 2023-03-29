package psclient_test

import (
	"encoding/json"
	"errors"

	"code.cloudfoundry.org/cf-networking-helpers/fakes"
	"code.cloudfoundry.org/policy-server/psclient"
	. "github.com/onsi/ginkgo/v2"
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
				Rules: []psclient.DestinationRule{
					{
						Protocol: "tcp",
						IPs:      "1.2.3.4-1.2.3.5",
						Ports:    "8080-9090",
					},
					{
						Protocol: "tcp",
						IPs:      "10.20.30.40-10.20.30.50",
						Ports:    "80-90",
					},
				},
			}

			destination2 = psclient.Destination{
				Name:        "bark-dest",
				Description: "dogs drool",
				Rules: []psclient.DestinationRule{
					{
						Protocol: "tcp",
						IPs:      "2.2.3.4-2.2.3.5",
						Ports:    "8081-9091",
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
						"destinations": [{
							 "id": "guid-received-from-server-1",
					     "name": "name-received-from-server",
					     "description": "description-received-from-server",
							 "rules": [{
								 "ips": "8.8.8.8-8.8.8.8",
								 "ports": "8080-8080",
								 "protocol": "tcp"
							 }, {
								 "ips": "9.9.9.9-9.9.9.9",
								 "ports": "80-80",
								 "protocol": "tcp"
							 }]
					  },
						{
							 "id": "guid-received-from-server-2",
					     "name": "name-received-from-server",
					     "description": "description-received-from-server",
							 "rules": [{
								 "ips": "8.8.8.8-8.8.8.8",
								 "ports": "8080-8080",
								 "protocol": "tcp"
							 }]
					  }]
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
						Rules: []psclient.DestinationRule{
							{
								IPs:      "8.8.8.8-8.8.8.8",
								Ports:    "8080-8080",
								Protocol: "tcp",
							},
							{
								IPs:      "9.9.9.9-9.9.9.9",
								Ports:    "80-80",
								Protocol: "tcp",
							},
						},
					},
					{
						GUID:        "guid-received-from-server-2",
						Name:        "name-received-from-server",
						Description: "description-received-from-server",
						Rules: []psclient.DestinationRule{
							{
								IPs:      "8.8.8.8-8.8.8.8",
								Ports:    "8080-8080",
								Protocol: "tcp",
							},
						},
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
						Name:        "destinationObject",
						Description: "description",
						GUID:        "response-dest-guid",
						Rules: []psclient.DestinationRule{
							{
								Protocol: "tcp",
								IPs:      "1.1.1.1-1.1.1.5",
								Ports:    "1234-2345",
							},
						},
					}},
				}
			})

			JustBeforeEach(func() {
				destinationRespBytes, err := json.Marshal(destinationResp)
				Expect(err).NotTo(HaveOccurred())

				jsonClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					json.Unmarshal(destinationRespBytes, respData)
					return nil
				}

				destinationToDelete = psclient.Destination{GUID: "dest-guid"}
			})

			Context("when nothing is deleted", func() {
				BeforeEach(func() {
					destinationResp = psclient.DestinationList{
						Destinations: []psclient.Destination{},
					}
				})
				It("returns an empty array", func() {
					deletedDestinations, err := client.DeleteDestination(token, destinationToDelete)
					Expect(err).NotTo(HaveOccurred())
					Expect(deletedDestinations).To(HaveLen(0))
				})
			})

			It("deletes destination", func() {
				deletedDestinations, err := client.DeleteDestination(token, destinationToDelete)
				Expect(err).NotTo(HaveOccurred())

				Expect(deletedDestinations[0]).To(Equal(destinationResp.Destinations[0]))

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
})
