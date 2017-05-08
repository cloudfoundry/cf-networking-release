package cc_client_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"policy-server/cc_client"
	"policy-server/cc_client/fixtures"
	"policy-server/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/go-db-helpers/fakes"
	"code.cloudfoundry.org/go-db-helpers/json_client"
	"code.cloudfoundry.org/lager/lagertest"
)

var _ = Describe("Client", func() {
	var (
		client         *cc_client.Client
		fakeJSONClient *fakes.JSONClient
		logger         *lagertest.TestLogger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		fakeJSONClient = &fakes.JSONClient{}
		client = &cc_client.Client{
			JSONClient: fakeJSONClient,
			Logger:     logger,
		}
	})

	Describe("GetAllAppGUIDs", func() {
		Context("when there is a single page of app guids", func() {
			BeforeEach(func() {
				fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					_ = json.Unmarshal([]byte(fixtures.AppsV3), respData)
					return nil
				}
			})

			It("returns the app guids", func() {
				apps, err := client.GetAllAppGUIDs("some-token")
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeJSONClient.DoCallCount()).To(Equal(1))

				method, route, reqData, _, token := fakeJSONClient.DoArgsForCall(0)

				Expect(method).To(Equal("GET"))
				Expect(route).To(Equal("/v3/apps"))
				Expect(reqData).To(BeNil())
				Expect(token).To(Equal("bearer some-token"))

				Expect(apps).To(Equal(map[string]struct{}{
					"live-app-1-guid": struct{}{},
					"live-app-2-guid": struct{}{},
					"live-app-3-guid": struct{}{},
					"live-app-4-guid": struct{}{},
					"live-app-5-guid": struct{}{},
				}))
			})
		})

		Context("when there are multiple pages", func() {
			BeforeEach(func() {
				fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					if route == "/v3/apps?page=2&per_page=1" {
						json.Unmarshal([]byte(fixtures.AppsV3MultiplePagesPg2), respData)
					} else if route == "/v3/apps?page=3&per_page=1" {
						json.Unmarshal([]byte(fixtures.AppsV3MultiplePagesPg3), respData)
					} else {
						json.Unmarshal([]byte(fixtures.AppsV3MultiplePages), respData)
					}
					return nil
				}
			})

			It("returns all the app guids", func() {
				apps, err := client.GetAllAppGUIDs("some-token")
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeJSONClient.DoCallCount()).To(Equal(3))

				method, route, reqData, _, token := fakeJSONClient.DoArgsForCall(0)

				Expect(method).To(Equal("GET"))
				Expect(route).To(Equal("/v3/apps"))
				Expect(reqData).To(BeNil())
				Expect(token).To(Equal("bearer some-token"))

				method, route, reqData, _, token = fakeJSONClient.DoArgsForCall(1)

				Expect(method).To(Equal("GET"))
				Expect(route).To(Equal("/v3/apps?page=2&per_page=1"))
				Expect(reqData).To(BeNil())
				Expect(token).To(Equal("bearer some-token"))

				method, route, reqData, _, token = fakeJSONClient.DoArgsForCall(2)

				Expect(method).To(Equal("GET"))
				Expect(route).To(Equal("/v3/apps?page=3&per_page=1"))
				Expect(reqData).To(BeNil())
				Expect(token).To(Equal("bearer some-token"))

				Expect(apps).To(Equal(map[string]struct{}{
					"live-app-1-guid": struct{}{},
					"live-app-2-guid": struct{}{},
					"live-app-3-guid": struct{}{},
				}))
			})
		})

		Context("when the json client returns an error", func() {
			BeforeEach(func() {
				fakeJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns the error", func() {
				_, err := client.GetAllAppGUIDs("some-token")
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})
	})

	Describe("GetLiveAppGUIDs", func() {
		BeforeEach(func() {
			fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				_ = json.Unmarshal([]byte(fixtures.AppsV3LiveAppGUIDs), respData)
				return nil
			}
		})

		It("Returns the app guids", func() {
			appGUIDs, err := client.GetLiveAppGUIDs("some-token", []string{"live-app-1-guid", "live-app-2-guid"})
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeJSONClient.DoCallCount()).To(Equal(1))

			method, route, reqData, _, token := fakeJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v3/apps?guids=live-app-1-guid%2Clive-app-2-guid&per_page=2"))
			Expect(reqData).To(BeNil())
			Expect(token).To(Equal("bearer some-token"))

			Expect(appGUIDs).To(Equal(map[string]struct{}{
				"live-app-1-guid": struct{}{},
				"live-app-2-guid": struct{}{},
			}))
		})

		Context("when the json client returns an error", func() {
			BeforeEach(func() {
				fakeJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns the error", func() {
				_, err := client.GetLiveAppGUIDs("some-token", []string{})
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})

		Context("when there are multiple pages", func() {
			BeforeEach(func() {
				fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					_ = json.Unmarshal([]byte(fixtures.AppsV3MultiplePages), respData)
					return nil
				}
			})

			It("should immediately return an error", func() {
				_, err := client.GetLiveAppGUIDs("some-token", []string{})
				Expect(err).To(MatchError("pagination support not yet implemented"))
			})
		})
	})

	Describe("GetSpaceGUIDs", func() {
		BeforeEach(func() {
			fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				_ = json.Unmarshal([]byte(fixtures.AppsV3), respData)
				return nil
			}
		})

		It("Returns the space guids", func() {
			spaceGUIDs, err := client.GetSpaceGUIDs("some-token", []string{"live-app-1-guid", "live-app-2-guid"})
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeJSONClient.DoCallCount()).To(Equal(1))

			method, route, reqData, _, token := fakeJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v3/apps?guids=live-app-1-guid%2Clive-app-2-guid"))
			Expect(reqData).To(BeNil())
			Expect(token).To(Equal("bearer some-token"))

			Expect(spaceGUIDs).To(ConsistOf([]string{"space-1-guid", "space-2-guid", "space-3-guid"}))
		})

		Context("when called with an empty list of app GUIDs", func() {
			It("returns an empty slice of space guids", func() {
				spaceGUIDs, err := client.GetSpaceGUIDs("some-token", []string{})
				Expect(err).NotTo(HaveOccurred())
				Expect(spaceGUIDs).To(BeEmpty())
			})
		})

		Context("when called with nil list of app GUIDs", func() {
			It("returns an empty slice of space guids", func() {
				spaceGUIDs, err := client.GetSpaceGUIDs("some-token", nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(spaceGUIDs).To(BeEmpty())
			})
		})

		Context("when the json client returns an error", func() {
			BeforeEach(func() {
				fakeJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns a helpful error", func() {
				_, err := client.GetSpaceGUIDs("some-token", []string{"foo"})
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})
	})

	Describe("GetSpace", func() {
		BeforeEach(func() {
			fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				_ = json.Unmarshal([]byte(fixtures.Space), respData)
				return nil
			}
		})

		It("returns the space with the matching GUID", func() {
			space := models.Space{
				Name:    "name-2064",
				OrgGUID: "6e1ca5aa-55f1-4110-a97f-1f3473e771b9",
			}

			matchingSpace, err := client.GetSpace("some-token", "some-space-guid")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeJSONClient.DoCallCount()).To(Equal(1))

			method, route, reqData, _, token := fakeJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v2/spaces/some-space-guid"))
			Expect(reqData).To(BeNil())
			Expect(token).To(Equal("bearer some-token"))

			Expect(matchingSpace).To(Equal(&space))
		})

		Context("when the json client returns an error", func() {
			BeforeEach(func() {
				fakeJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns a helpful error", func() {
				_, err := client.GetSpace("some-token", "some-space-guid")
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})

		Context("if the response status code is a 404", func() {
			BeforeEach(func() {
				fakeJSONClient.DoReturns(&json_client.HttpResponseCodeError{
					StatusCode: 404,
					Message:    "not found",
				})
			})

			It("returns nil", func() {
				space, err := client.GetSpace("some-token", "some-space-guid")
				Expect(err).NotTo(HaveOccurred())
				Expect(space).To(BeNil())
			})
		})

		Context("if the response status code is not 200 or 404", func() {
			BeforeEach(func() {
				fakeJSONClient.DoReturns(&json_client.HttpResponseCodeError{
					StatusCode: http.StatusTeapot,
					Message:    "i am a teapot",
				})
			})

			It("returns a helpful error", func() {
				_, err := client.GetSpace("some-token", "some-space-guid")
				Expect(err).To(MatchError(ContainSubstring("json client do: http status 418: i am a teapot")))
			})
		})
	})

	Describe("GetAppSpaces", func() {
		appGUIDs := []string{}
		expectedAppSpaces := map[string]string{
			"live-app-1-guid": "space-1-guid",
			"live-app-2-guid": "space-1-guid",
			"live-app-3-guid": "space-2-guid",
			"live-app-4-guid": "space-2-guid",
			"live-app-5-guid": "space-3-guid",
		}
		BeforeEach(func() {
			for key, _ := range expectedAppSpaces {
				appGUIDs = append(appGUIDs, key)
			}
			fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				_ = json.Unmarshal([]byte(fixtures.AppsV3), respData)
				return nil
			}
		})

		It("returns the map from app to its space", func() {
			appSpaceMap, err := client.GetAppSpaces("some-token", appGUIDs)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeJSONClient.DoCallCount()).To(Equal(1))

			method, route, reqData, _, token := fakeJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(ContainSubstring("/v3/apps?guids="))
			for appGuid, _ := range expectedAppSpaces {
				Expect(route).To(ContainSubstring(appGuid))
			}
			Expect(reqData).To(BeNil())
			Expect(token).To(Equal("bearer some-token"))

			Expect(appSpaceMap).To(Equal(expectedAppSpaces))
		})

		Context("when the list of app GUIDs is empty", func() {
			It("returns an empty slice", func() {
				appSpaceMap, err := client.GetAppSpaces("some-token", []string{})
				Expect(err).NotTo(HaveOccurred())
				Expect(appSpaceMap).To(BeEmpty())
			})
		})

		Context("when the json client returns an error", func() {
			BeforeEach(func() {
				fakeJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns a helpful error", func() {
				_, err := client.GetAppSpaces("some-token", []string{"some-guid"})
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})
	})

	Describe("GetUserSpaces", func() {
		BeforeEach(func() {
			fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				_ = json.Unmarshal([]byte(fixtures.UserSpaces), respData)
				return nil
			}
		})

		It("returns the list of spaces a user has access to", func() {
			userSpaces, err := client.GetUserSpaces("some-token", "some-user-guid")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeJSONClient.DoCallCount()).To(Equal(1))

			method, route, reqData, _, token := fakeJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v2/users/some-user-guid/spaces"))
			Expect(reqData).To(BeNil())
			Expect(token).To(Equal("bearer some-token"))

			Expect(userSpaces).To(Equal(map[string]struct{}{
				"space-1-guid": struct{}{},
				"space-2-guid": struct{}{},
			}))
		})

		Context("when the json client returns an error", func() {
			BeforeEach(func() {
				fakeJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns a helpful error", func() {
				_, err := client.GetUserSpaces("some-token", "some-user-guid")
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})
	})

	Describe("GetUserSpace", func() {
		space := models.Space{
			Name:    "some-space-name",
			OrgGUID: "some-org-guid",
		}
		BeforeEach(func() {
			fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				_ = json.Unmarshal([]byte(fixtures.UserSpace), respData)
				return nil
			}
		})

		It("returns the matching spaces for the user", func() {
			matchingSpace, err := client.GetUserSpace("some-token", "some-developer-guid", space)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeJSONClient.DoCallCount()).To(Equal(1))

			method, route, reqData, _, token := fakeJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v2/spaces?q=developer_guid%3Asome-developer-guid&q=name%3Asome-space-name&q=organization_guid%3Asome-org-guid"))
			Expect(reqData).To(BeNil())
			Expect(token).To(Equal("bearer some-token"))

			Expect(matchingSpace).To(Equal(&space))
		})

		Context("when the user has no spaces", func() {
			BeforeEach(func() {
				fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					_ = json.Unmarshal([]byte(fixtures.UserSpaceEmpty), respData)
					return nil
				}
			})

			It("returns nil", func() {
				space, err := client.GetUserSpace("some-token", "some-developer-guid", space)
				Expect(err).NotTo(HaveOccurred())
				Expect(space).To(BeNil())
			})
		})

		Context("when more than one space is returned", func() {
			BeforeEach(func() {
				fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					_ = json.Unmarshal([]byte(fixtures.Spaces), respData)
					return nil
				}
			})

			It("returns an error", func() {
				_, err := client.GetUserSpace("some-token", "some-developer-guid", space)
				Expect(err).To(MatchError("found more than one matching space"))
			})
		})

		Context("when the json client returns an error", func() {
			BeforeEach(func() {
				fakeJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns a helpful error", func() {
				_, err := client.GetUserSpace("some-token", "some-developer-guid", space)
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})
	})
})
