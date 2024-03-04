package cc_client_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/fakes"
	"code.cloudfoundry.org/cf-networking-helpers/json_client"
	"code.cloudfoundry.org/lager/v3/lagertest"
	"code.cloudfoundry.org/policy-server/cc_client"
	"code.cloudfoundry.org/policy-server/cc_client/fixtures"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		client                 *cc_client.Client
		fakeExternalJSONClient *fakes.JSONClient
		fakeInternalJSONClient *fakes.JSONClient
		logger                 *lagertest.TestLogger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		fakeExternalJSONClient = &fakes.JSONClient{}
		fakeInternalJSONClient = &fakes.JSONClient{}
		client = &cc_client.Client{
			ExternalJSONClient: fakeExternalJSONClient,
			InternalJSONClient: fakeInternalJSONClient,
			Logger:             logger,
		}
	})

	Describe("GetAllAppGUIDs", func() {
		Context("when there is a single page of app guids", func() {
			BeforeEach(func() {
				fakeExternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					_ = json.Unmarshal([]byte(fixtures.AppsV3), respData)
					return nil
				}
			})

			It("returns the app guids", func() {
				apps, err := client.GetAllAppGUIDs("some-token")
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeExternalJSONClient.DoCallCount()).To(Equal(1))

				method, route, reqData, _, token := fakeExternalJSONClient.DoArgsForCall(0)

				Expect(method).To(Equal("GET"))
				Expect(route).To(Equal("/v3/apps"))
				Expect(reqData).To(BeNil())
				Expect(token).To(Equal("bearer some-token"))

				Expect(apps).To(Equal(map[string]struct{}{
					"live-app-1-guid": {},
					"live-app-2-guid": {},
					"live-app-3-guid": {},
					"live-app-4-guid": {},
					"live-app-5-guid": {},
				}))
			})
		})

		Context("when there are multiple pages", func() {
			BeforeEach(func() {
				fakeExternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
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

				Expect(fakeExternalJSONClient.DoCallCount()).To(Equal(3))

				method, route, reqData, _, token := fakeExternalJSONClient.DoArgsForCall(0)

				Expect(method).To(Equal("GET"))
				Expect(route).To(Equal("/v3/apps"))
				Expect(reqData).To(BeNil())
				Expect(token).To(Equal("bearer some-token"))

				method, route, reqData, _, token = fakeExternalJSONClient.DoArgsForCall(1)

				Expect(method).To(Equal("GET"))
				Expect(route).To(Equal("/v3/apps?page=2&per_page=1"))
				Expect(reqData).To(BeNil())
				Expect(token).To(Equal("bearer some-token"))

				method, route, reqData, _, token = fakeExternalJSONClient.DoArgsForCall(2)

				Expect(method).To(Equal("GET"))
				Expect(route).To(Equal("/v3/apps?page=3&per_page=1"))
				Expect(reqData).To(BeNil())
				Expect(token).To(Equal("bearer some-token"))

				Expect(apps).To(Equal(map[string]struct{}{
					"live-app-1-guid": {},
					"live-app-2-guid": {},
					"live-app-3-guid": {},
				}))
			})
		})

		Context("when the json client returns an error", func() {
			BeforeEach(func() {
				fakeExternalJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns the error", func() {
				_, err := client.GetAllAppGUIDs("some-token")
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})
	})

	Describe("GetLiveAppGUIDs", func() {
		BeforeEach(func() {
			fakeExternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				_ = json.Unmarshal([]byte(fixtures.AppsV3LiveAppGUIDs), respData)
				return nil
			}
		})

		It("Returns the app guids", func() {
			appGUIDs, err := client.GetLiveAppGUIDs("some-token", []string{"live-app-1-guid", "live-app-2-guid"})
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeExternalJSONClient.DoCallCount()).To(Equal(1))

			method, route, reqData, _, token := fakeExternalJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v3/apps?guids=live-app-1-guid%2Clive-app-2-guid&per_page=2"))
			Expect(reqData).To(BeNil())
			Expect(token).To(Equal("bearer some-token"))

			Expect(appGUIDs).To(Equal(map[string]struct{}{
				"live-app-1-guid": {},
				"live-app-2-guid": {},
			}))
		})

		Context("when the json client returns an error", func() {
			BeforeEach(func() {
				fakeExternalJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns the error", func() {
				_, err := client.GetLiveAppGUIDs("some-token", []string{})
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})

		Context("when there are multiple pages", func() {
			BeforeEach(func() {
				fakeExternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
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

	Describe("GetLiveSpaceGUIDs", func() {
		var (
			passedToken string
			passedRoute string
		)

		BeforeEach(func() {
			fakeExternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				passedToken = token
				passedRoute = route
				_ = json.Unmarshal([]byte(fixtures.SpaceV3LiveSpaces), respData)
				return nil
			}
		})

		It("returns the live space guids filtered by given space guids", func() {
			liveSpaceGUIDs, err := client.GetLiveSpaceGUIDs("some-token", []string{"live-space-1-guid", "live-space-2-guid", "dead-space-1-guid"})
			Expect(err).NotTo(HaveOccurred())
			Expect(liveSpaceGUIDs).To(Equal(map[string]struct{}{
				"live-space-1-guid": {},
				"live-space-2-guid": {},
			}))

			Expect(passedToken).To(Equal("bearer some-token"))
			Expect(passedRoute).To(Equal("/v3/spaces?guids=live-space-1-guid%2Clive-space-2-guid%2Cdead-space-1-guid&per_page=4"))
		})

		Context("when the json client returns an error", func() {
			BeforeEach(func() {
				fakeExternalJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns the error", func() {
				_, err := client.GetLiveSpaceGUIDs("some-token", []string{})
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})

		Context("when there are multiple pages", func() {
			BeforeEach(func() {
				fakeExternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					_ = json.Unmarshal([]byte(fixtures.SpaceV3MultiplePages), respData)
					return nil
				}
			})

			It("should immediately return an error", func() {
				_, err := client.GetLiveSpaceGUIDs("some-token", []string{})
				Expect(err).To(MatchError("pagination support not yet implemented"))
			})
		})
	})

	Describe("GetSpaceGUIDs", func() {
		BeforeEach(func() {
			fakeExternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				_ = json.Unmarshal([]byte(fixtures.AppsV3), respData)
				return nil
			}
		})

		It("Returns the space guids", func() {
			spaceGUIDs, err := client.GetSpaceGUIDs("some-token", []string{"live-app-1-guid", "live-app-2-guid"})
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeExternalJSONClient.DoCallCount()).To(Equal(1))

			method, route, reqData, _, token := fakeExternalJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v3/apps?guids=live-app-1-guid%2Clive-app-2-guid&per_page=2"))
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
				fakeExternalJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns a helpful error", func() {
				_, err := client.GetSpaceGUIDs("some-token", []string{"foo"})
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})
	})

	Describe("GetSpace", func() {
		BeforeEach(func() {
			fakeExternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				_ = json.Unmarshal([]byte(fixtures.Space), respData)
				return nil
			}
		})

		It("returns the space with the matching GUID", func() {
			space := cc_client.SpaceResponse{
				Entity: cc_client.SpaceEntity{
					Name:             "name-2064",
					OrganizationGUID: "6e1ca5aa-55f1-4110-a97f-1f3473e771b9",
				},
			}

			matchingSpace, err := client.GetSpace("some-token", "some-space-guid")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeExternalJSONClient.DoCallCount()).To(Equal(1))

			method, route, reqData, _, token := fakeExternalJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v2/spaces/some-space-guid"))
			Expect(reqData).To(BeNil())
			Expect(token).To(Equal("bearer some-token"))

			Expect(matchingSpace).To(Equal(&space))
		})

		Context("when the json client returns an error", func() {
			BeforeEach(func() {
				fakeExternalJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns a helpful error", func() {
				_, err := client.GetSpace("some-token", "some-space-guid")
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})

		Context("if the response status code is a 404", func() {
			BeforeEach(func() {
				fakeExternalJSONClient.DoReturns(&json_client.HttpResponseCodeError{
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
				fakeExternalJSONClient.DoReturns(&json_client.HttpResponseCodeError{
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
		var (
			appGUIDs          []string
			expectedAppSpaces map[string]string
		)
		BeforeEach(func() {
			appGUIDs = []string{
				"live-app-1-guid",
				"live-app-2-guid",
				"live-app-3-guid",
				"live-app-4-guid",
				"live-app-5-guid",
			}
			expectedAppSpaces = map[string]string{
				"live-app-1-guid": "space-1-guid",
				"live-app-2-guid": "space-1-guid",
				"live-app-3-guid": "space-2-guid",
				"live-app-4-guid": "space-2-guid",
				"live-app-5-guid": "space-3-guid",
			}
			fakeExternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				_ = json.Unmarshal([]byte(fixtures.AppsV3), respData)
				return nil
			}
		})

		It("returns the map from app to its space", func() {
			appSpaceMap, err := client.GetAppSpaces("some-token", appGUIDs)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeExternalJSONClient.DoCallCount()).To(Equal(1))

			method, route, reqData, _, token := fakeExternalJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(ContainSubstring("/v3/apps?guids="))
			for appGuid := range expectedAppSpaces {
				Expect(route).To(ContainSubstring(appGuid))
			}
			Expect(route).To(ContainSubstring("per_page=5"))
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
				fakeExternalJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns a helpful error", func() {
				_, err := client.GetAppSpaces("some-token", []string{"some-guid"})
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})

		Context("when there are multiple pages", func() {
			BeforeEach(func() {
				fakeExternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					_ = json.Unmarshal([]byte(fixtures.AppsV3MultiplePages), respData)
					return nil
				}
			})

			It("should immediately return an error", func() {
				_, err := client.GetAppSpaces("some-token", []string{"some-guid"})
				Expect(err).To(MatchError("pagination support not yet implemented"))
			})
		})
	})

	Describe("GetSubjectSpaces", func() {
		BeforeEach(func() {
			responses := []string{
				fixtures.SubjectSpacesPage1,
				fixtures.SubjectSpacesPage2,
				fixtures.SubjectSpacesPage3,
			}
			fakeExternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				parsedRoute, err := url.Parse(route)
				Expect(err).NotTo(HaveOccurred())
				pageParameter := parsedRoute.Query().Get("page")
				if pageParameter == "" {
					pageParameter = "1"
				}
				page, err := strconv.Atoi(pageParameter)
				Expect(err).NotTo(HaveOccurred())
				response := responses[page-1]
				err = json.Unmarshal([]byte(response), respData)
				Expect(err).NotTo(HaveOccurred())
				return nil
			}
		})

		It("returns the list of spaces a subject has access to", func() {
			subjectSpaces, err := client.GetSubjectSpaces("some-token", "some-subject-id")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeExternalJSONClient.DoCallCount()).To(Equal(3))

			method, route, reqData, _, token := fakeExternalJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v2/users/some-subject-id/spaces?results-per-page=100"))
			Expect(reqData).To(BeNil())
			Expect(token).To(Equal("bearer some-token"))

			method, route, reqData, _, token = fakeExternalJSONClient.DoArgsForCall(1)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v2/users/some-subject-id/spaces?order-direction=asc&page=2&results-per-page=1"))
			Expect(reqData).To(BeNil())
			Expect(token).To(Equal("bearer some-token"))

			method, route, reqData, _, token = fakeExternalJSONClient.DoArgsForCall(2)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v2/users/some-subject-id/spaces?order-direction=asc&page=3&results-per-page=1"))
			Expect(reqData).To(BeNil())
			Expect(token).To(Equal("bearer some-token"))

			Expect(subjectSpaces).To(Equal(map[string]struct{}{
				"space-1-guid": {},
				"space-2-guid": {},
				"space-3-guid": {},
			}))
		})

		Context("when the json client returns an error", func() {
			BeforeEach(func() {
				fakeExternalJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns a helpful error", func() {
				_, err := client.GetSubjectSpaces("some-token", "some-subject-id")
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})
	})

	Describe("GetSubjectSpace", func() {
		space := cc_client.SpaceResponse{
			Entity: cc_client.SpaceEntity{
				Name:             "some-space-name",
				OrganizationGUID: "some-org-guid",
			},
		}
		BeforeEach(func() {
			fakeExternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				_ = json.Unmarshal([]byte(fixtures.SubjectSpace), respData)
				return nil
			}
		})

		It("returns the matching spaces for the subject", func() {
			matchingSpace, err := client.GetSubjectSpace("some-token", "some-subject-id", space)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeExternalJSONClient.DoCallCount()).To(Equal(1))

			method, route, reqData, _, token := fakeExternalJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v2/spaces?q=developer_guid%3Asome-subject-id&q=name%3Asome-space-name&q=organization_guid%3Asome-org-guid"))
			Expect(reqData).To(BeNil())
			Expect(token).To(Equal("bearer some-token"))

			Expect(matchingSpace.Entity).To(Equal(space.Entity))
		})

		Context("when the subject has no spaces", func() {
			BeforeEach(func() {
				fakeExternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					_ = json.Unmarshal([]byte(fixtures.SubjectSpaceEmpty), respData)
					return nil
				}
			})

			It("returns nil", func() {
				space, err := client.GetSubjectSpace("some-token", "some-subject-id", space)
				Expect(err).NotTo(HaveOccurred())
				Expect(space).To(BeNil())
			})
		})

		Context("when more than one space is returned", func() {
			BeforeEach(func() {
				fakeExternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					_ = json.Unmarshal([]byte(fixtures.Spaces), respData)
					return nil
				}
			})

			It("returns an error", func() {
				_, err := client.GetSubjectSpace("some-token", "some-subject-id", space)
				Expect(err).To(MatchError("found more than one matching space"))
			})
		})

		Context("when the json client returns an error", func() {
			BeforeEach(func() {
				fakeExternalJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns a helpful error", func() {
				_, err := client.GetSubjectSpace("some-token", "some-subject-id", space)
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})
	})

	Describe("GetSecurityGroupsWithPage", func() {
		BeforeEach(func() {
			fakeExternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				_ = json.Unmarshal([]byte(`{"resources":[{"guid": "meow"}]}`), respData)
				return nil
			}
		})

		It("returns the response from the JSON Client", func() {
			response, err := client.GetSecurityGroupsWithPage("token", 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())
			Expect(response.Resources).NotTo(BeEmpty())
			Expect(response.Resources[0].GUID).To(Equal("meow"))
		})

		It("makes a JSONClient call with the provided page number", func() {
			_, err := client.GetSecurityGroupsWithPage("token", 1)
			Expect(err).NotTo(HaveOccurred())

			_, route, _, _, _ := fakeExternalJSONClient.DoArgsForCall(0)
			Expect(route).To(Equal("/v3/security_groups?per_page=5000&page=1"))

			_, err = client.GetSecurityGroupsWithPage("token", 2)
			Expect(err).NotTo(HaveOccurred())

			_, route, _, _, _ = fakeExternalJSONClient.DoArgsForCall(1)
			Expect(route).To(Equal("/v3/security_groups?per_page=5000&page=2"))
		})

		It("makes a JSONClient call with the provided page number", func() {
			_, err := client.GetSecurityGroupsWithPage("token", 1)
			Expect(err).NotTo(HaveOccurred())
			_, _, _, _, token := fakeExternalJSONClient.DoArgsForCall(0)
			Expect(token).To(Equal("bearer token"))
		})
	})

	Describe("GetSecurityGroupsLastUpdate", func() {
		BeforeEach(func() {
			fakeInternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				_ = json.Unmarshal([]byte(`{"last_update":"1971-01-01T00:00:00Z"}`), respData)
				return nil
			}
		})

		It("gets latest update time", func() {
			latestUpdateTimeResponse, err := client.GetSecurityGroupsLastUpdate("some-token")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeInternalJSONClient.DoCallCount()).To(Equal(1))

			method, route, reqData, _, _ := fakeInternalJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/internal/v4/asg_latest_update"))
			Expect(reqData).To(BeNil())
			timestamp, err := time.Parse(time.RFC3339, "1971-01-01T00:00:00Z")
			Expect(err).NotTo(HaveOccurred())
			Expect(latestUpdateTimeResponse).To(Equal(timestamp))
		})

		Context("when the endpoint does not exist", func() {
			BeforeEach(func() {
				fakeInternalJSONClient.DoReturns(&json_client.HttpResponseCodeError{
					StatusCode: 404,
					Message:    "not found",
				})
			})

			It("returns zero time", func() {
				latestUpdateTimeResponse, err := client.GetSecurityGroupsLastUpdate("some-token")
				Expect(err).NotTo(HaveOccurred())
				Expect(latestUpdateTimeResponse.IsZero()).To(BeTrue())
			})
		})

		Context("when the timestamp can't be parsed", func() {
			BeforeEach(func() {
				fakeInternalJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					_ = json.Unmarshal([]byte(`{"last_update":"meow-meow"}`), respData)
					return nil
				}
			})

			It("returns zero time and an error", func() {
				latestUpdateTimeResponse, err := client.GetSecurityGroupsLastUpdate("some-token")
				Expect(latestUpdateTimeResponse.IsZero()).To(BeTrue())
				Expect(err).To(MatchError(ContainSubstring("failed parsing last_update from cloud controller: 'meow-meow'")))
			})
		})
	})

	Describe("GetSecurityGroups", func() {
		var (
			token string
		)

		BeforeEach(func() {
			token = "some-uaa-token"
			stubDefaultLastUpdateRequest(fakeInternalJSONClient)
		})

		Context("when there are no security groups", func() {
			It("returns an empty set of security groups", func() {
				Expect(client.GetSecurityGroups(token)).To(Equal([]cc_client.SecurityGroupResource{}))
			})
		})

		Context("when there is an error fetching a page of security groups", func() {
			BeforeEach(func() {
				fakeExternalJSONClient.DoStub = func(method, route string, req, resp interface{}, token string) error {
					switch fakeExternalJSONClient.DoCallCount() {
					case 1:
						return errors.New("security-groups-error")
					case 2:
						loadSecurityGroupsResponseIntoObject([]cc_client.SecurityGroupResource{}, resp)
						return nil
					default:
						panic("Too many calls to fetch security groups")
					}
				}
			})

			It("bails on that pagination attempt and retries", func() {
				securityGroups, err := client.GetSecurityGroups(token)
				Expect(securityGroups).To(BeEmpty())
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeExternalJSONClient.DoCallCount()).To(Equal(2))

				method, route, req, _, token := fakeExternalJSONClient.DoArgsForCall(0)
				Expect(method).To(Equal("GET"))
				Expect(route).To(ContainSubstring("security_groups"))
				Expect(route).To(ContainSubstring("page=1"))
				Expect(req).To(BeNil())
				Expect(token).To(Equal("bearer some-uaa-token"))

				method, route, req, _, token = fakeExternalJSONClient.DoArgsForCall(1)
				Expect(method).To(Equal("GET"))
				Expect(route).To(ContainSubstring("security_groups"))
				Expect(route).To(ContainSubstring("page=1"))
				Expect(req).To(BeNil())
				Expect(token).To(Equal("bearer some-uaa-token"))
			})
		})

		Context("when there is an error fetching the initial last_update time", func() {
			BeforeEach(func() {
				timestamp := time.Now()
				fakeInternalJSONClient.DoStub = func(method, route string, req, resp interface{}, token string) error {
					switch fakeInternalJSONClient.DoCallCount() {
					case 1:
						return errors.New("last-update-error")
					default:
						loadLastUpdateResponseIntoObject(timestamp, resp)
						return nil
					}
				}
			})

			It("bails on that pagination attempt and retries", func() {
				securityGroups, err := client.GetSecurityGroups(token)
				Expect(securityGroups).To(BeEmpty())
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeInternalJSONClient.DoCallCount()).To(Equal(3))
				//TODO: Make stronger assertion on parameters passed to Do method
			})
		})

		Context("when the CC API returns a single page of security groups", func() {
			BeforeEach(func() {
				securityGroups := []cc_client.SecurityGroupResource{
					cc_client.SecurityGroupResource{GUID: "1"},
					cc_client.SecurityGroupResource{GUID: "2"},
					cc_client.SecurityGroupResource{GUID: "3"},
				}

				stubSecurityGroupRequestWith(fakeExternalJSONClient, securityGroups, nil)
			})

			It("returns the security groups as a list of security group resources", func() {
				Expect(client.GetSecurityGroups(token)).To(Equal([]cc_client.SecurityGroupResource{
					cc_client.SecurityGroupResource{GUID: "1"},
					cc_client.SecurityGroupResource{GUID: "2"},
					cc_client.SecurityGroupResource{GUID: "3"},
				}))
			})
		})

		Context("when the CC API returns paginated results", func() {
			var (
				response0, response1, response2 cc_client.GetSecurityGroupsResponse
				stubResponses                   []cc_client.GetSecurityGroupsResponse
			)

			BeforeEach(func() {
				response0 = cc_client.GetSecurityGroupsResponse{
					Pagination: cc_client.Pagination{
						TotalPages: 3,
						Next:       cc_client.Href{Href: "url-for-page-1"},
					},
					Resources: []cc_client.SecurityGroupResource{
						cc_client.SecurityGroupResource{GUID: "1"},
						cc_client.SecurityGroupResource{GUID: "2"},
						cc_client.SecurityGroupResource{GUID: "3"},
					},
				}

				response1 = cc_client.GetSecurityGroupsResponse{
					Pagination: cc_client.Pagination{
						TotalPages: 3,
						Next:       cc_client.Href{Href: "url-for-page-2"},
						Previous:   cc_client.Href{Href: "url-for-page-1"},
					},
					Resources: []cc_client.SecurityGroupResource{
						cc_client.SecurityGroupResource{GUID: "4"},
						cc_client.SecurityGroupResource{GUID: "5"},
						cc_client.SecurityGroupResource{GUID: "6"},
					},
				}

				response2 = cc_client.GetSecurityGroupsResponse{
					Pagination: cc_client.Pagination{
						TotalPages: 3,
						Previous:   cc_client.Href{Href: "url-for-page-1"},
					},
					Resources: []cc_client.SecurityGroupResource{
						cc_client.SecurityGroupResource{GUID: "7"},
						cc_client.SecurityGroupResource{GUID: "8"},
						cc_client.SecurityGroupResource{GUID: "9"},
					},
				}

				stubResponses = []cc_client.GetSecurityGroupsResponse{response0, response1, response2}
				fakeExternalJSONClient.DoStub = func(method, route string, req, resp interface{}, token string) error {
					jsonBytes, err := json.Marshal(stubResponses[fakeExternalJSONClient.DoCallCount()-1])
					Expect(err).NotTo(HaveOccurred())

					err = json.Unmarshal(jsonBytes, resp)
					Expect(err).NotTo(HaveOccurred())
					return nil
				}
			})

			Context("and the CC last_update time does not change between fetching pages", func() {
				BeforeEach(func() {
					stubDefaultLastUpdateRequest(fakeInternalJSONClient)
				})

				It("requests the next page of results", func() {
					Expect(client.GetSecurityGroups(token)).To(Equal([]cc_client.SecurityGroupResource{
						cc_client.SecurityGroupResource{GUID: "1"},
						cc_client.SecurityGroupResource{GUID: "2"},
						cc_client.SecurityGroupResource{GUID: "3"},
						cc_client.SecurityGroupResource{GUID: "4"},
						cc_client.SecurityGroupResource{GUID: "5"},
						cc_client.SecurityGroupResource{GUID: "6"},
						cc_client.SecurityGroupResource{GUID: "7"},
						cc_client.SecurityGroupResource{GUID: "8"},
						cc_client.SecurityGroupResource{GUID: "9"},
					}))

					Expect(fakeExternalJSONClient.DoCallCount()).To(Equal(3))

					_, route, _, _, token := fakeExternalJSONClient.DoArgsForCall(0)
					Expect(token).To(Equal("bearer some-uaa-token"))
					Expect(route).To(ContainSubstring("page=1"))

					_, route, _, _, token = fakeExternalJSONClient.DoArgsForCall(1)
					Expect(token).To(Equal("bearer some-uaa-token"))
					Expect(route).To(ContainSubstring("page=2"))

					_, route, _, _, token = fakeExternalJSONClient.DoArgsForCall(2)
					Expect(token).To(Equal("bearer some-uaa-token"))
					Expect(route).To(ContainSubstring("page=3"))

					Expect(fakeInternalJSONClient.DoCallCount()).To(Equal(4))
				})

				Context("when calls to CC continually fail", func() {
					BeforeEach(func() {
						// First try fails
						fakeInternalJSONClient.DoReturnsOnCall(0, errors.New("meow"))
						// Second try fails
						fakeInternalJSONClient.DoReturnsOnCall(1, errors.New("meow"))
						// Third try fails
						fakeInternalJSONClient.DoReturnsOnCall(2, errors.New("meow"))
					})

					It("returns an error", func() {
						securityGroups, err := client.GetSecurityGroups(token)
						Expect(securityGroups).To(BeEmpty())
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("Ran out of retry attempts"))
						Expect(err.Error()).To(ContainSubstring("meow"))
					})
				})

				Context("when there is an error fetching the last_update time", func() {
					BeforeEach(func() {
						timestamp := time.Now()
						failedCallIndex := 2 // Zero-indexed

						fakeInternalJSONClient.DoStub = func(method, route string, req, resp interface{}, token string) error {
							switch fakeInternalJSONClient.DoCallCount() - 1 { // DoCallCount it 1-indexed
							case failedCallIndex:
								// First try fails on fetching the timestamp for the last page
								return errors.New("last-update-error")
							default:
								loadLastUpdateResponseIntoObject(timestamp, resp)
								return nil
							}
						}

						fakeExternalJSONClient.DoStub = func(method, route string, req, resp interface{}, token string) error {
							responseNumber := fakeExternalJSONClient.DoCallCount() - 1 //DoCallCount is 1-indexed
							if responseNumber >= failedCallIndex {
								responseNumber = responseNumber - failedCallIndex
							}

							jsonBytes, err := json.Marshal(stubResponses[responseNumber])
							Expect(err).NotTo(HaveOccurred())

							err = json.Unmarshal(jsonBytes, resp)
							Expect(err).NotTo(HaveOccurred())
							return nil
						}
					})

					It("bails on that pagination attempt and retries", func() {
						securityGroups, err := client.GetSecurityGroups(token)
						Expect(securityGroups).To(Equal([]cc_client.SecurityGroupResource{
							cc_client.SecurityGroupResource{GUID: "1"},
							cc_client.SecurityGroupResource{GUID: "2"},
							cc_client.SecurityGroupResource{GUID: "3"},
							cc_client.SecurityGroupResource{GUID: "4"},
							cc_client.SecurityGroupResource{GUID: "5"},
							cc_client.SecurityGroupResource{GUID: "6"},
							cc_client.SecurityGroupResource{GUID: "7"},
							cc_client.SecurityGroupResource{GUID: "8"},
							cc_client.SecurityGroupResource{GUID: "9"},
						}))
						Expect(err).NotTo(HaveOccurred())
						Expect(fakeInternalJSONClient.DoCallCount()).To(Equal(7))
					})
				})

			})

			Context("and the CC last_update time changes between fetching pages", func() {
				BeforeEach(func() {
					timestamp1 := time.Now()
					timestamp2 := timestamp1.Add(1 * time.Minute)

					timestampChangedIndex := 2 //Zero-indexed

					// First try "fails" because the timestamp has been updated
					// Second try "succeeds" because the new timestamp persists through pagination
					fakeInternalJSONClient.DoStub = func(method, route string, req, resp interface{}, token string) error {
						Expect(fakeInternalJSONClient.DoCallCount()).To(BeNumerically("<=", 7))
						switch callCount := fakeInternalJSONClient.DoCallCount() - 1; { //DoCallCount is 1-indexed
						case callCount < timestampChangedIndex:
							loadLastUpdateResponseIntoObject(timestamp1, resp)
							return nil
						case callCount >= timestampChangedIndex:
							loadLastUpdateResponseIntoObject(timestamp2, resp)
							return nil
						default:
							panic("impossible state")
						}
					}

					fakeExternalJSONClient.DoStub = func(method, route string, req, resp interface{}, token string) error {
						responseNumber := fakeExternalJSONClient.DoCallCount() - 1 //DoCallCount is 1-indexed
						if responseNumber >= timestampChangedIndex {
							responseNumber = responseNumber - timestampChangedIndex
						}

						jsonBytes, err := json.Marshal(stubResponses[responseNumber])
						Expect(err).NotTo(HaveOccurred())

						err = json.Unmarshal(jsonBytes, resp)
						Expect(err).NotTo(HaveOccurred())
						return nil
					}
				})

				It("retries by starting from the beginning", func() {
					Expect(client.GetSecurityGroups(token)).To(Equal([]cc_client.SecurityGroupResource{
						cc_client.SecurityGroupResource{GUID: "1"},
						cc_client.SecurityGroupResource{GUID: "2"},
						cc_client.SecurityGroupResource{GUID: "3"},
						cc_client.SecurityGroupResource{GUID: "4"},
						cc_client.SecurityGroupResource{GUID: "5"},
						cc_client.SecurityGroupResource{GUID: "6"},
						cc_client.SecurityGroupResource{GUID: "7"},
						cc_client.SecurityGroupResource{GUID: "8"},
						cc_client.SecurityGroupResource{GUID: "9"},
					}))

					Expect(fakeExternalJSONClient.DoCallCount()).To(Equal(5))

					_, route, _, _, token := fakeExternalJSONClient.DoArgsForCall(0)
					Expect(token).To(Equal("bearer some-uaa-token"))
					Expect(route).To(ContainSubstring("page=1"))

					_, route, _, _, token = fakeExternalJSONClient.DoArgsForCall(1)
					Expect(token).To(Equal("bearer some-uaa-token"))
					Expect(route).To(ContainSubstring("page=2"))

					_, route, _, _, token = fakeExternalJSONClient.DoArgsForCall(2)
					Expect(token).To(Equal("bearer some-uaa-token"))
					Expect(route).To(ContainSubstring("page=1"))

					_, route, _, _, token = fakeExternalJSONClient.DoArgsForCall(3)
					Expect(token).To(Equal("bearer some-uaa-token"))
					Expect(route).To(ContainSubstring("page=2"))

					_, route, _, _, token = fakeExternalJSONClient.DoArgsForCall(4)
					Expect(token).To(Equal("bearer some-uaa-token"))
					Expect(route).To(ContainSubstring("page=3"))

					Expect(fakeInternalJSONClient.DoCallCount()).To(Equal(7))
				})
			})

			Context("and the CC last_update time changes between fetching pages more times than the retry limit", func() {
				BeforeEach(func() {
					timestamp1 := time.Now()
					timestamp2 := timestamp1.Add(1 * time.Minute)
					timestamp3 := timestamp1.Add(2 * time.Minute)
					timestamp4 := timestamp1.Add(3 * time.Minute)

					timestampChangedIndex1 := 2
					timestampChangedIndex2 := 5
					timestampChangedIndex3 := 8

					fakeInternalJSONClient.DoStub = func(method, route string, req, resp interface{}, token string) error {
						switch callCount := fakeInternalJSONClient.DoCallCount() - 1; { //DoCallCount is 1-indexed
						case callCount < timestampChangedIndex1:
							loadLastUpdateResponseIntoObject(timestamp1, resp)
							return nil
						case callCount <= timestampChangedIndex2:
							loadLastUpdateResponseIntoObject(timestamp2, resp)
							return nil
						case callCount <= timestampChangedIndex3:
							loadLastUpdateResponseIntoObject(timestamp3, resp)
							return nil
						default:
							loadLastUpdateResponseIntoObject(timestamp4, resp)
							return nil
						}
					}

					fakeExternalJSONClient.DoStub = func(method, route string, req, resp interface{}, token string) error {
						var response cc_client.GetSecurityGroupsResponse
						switch responseNumber := fakeExternalJSONClient.DoCallCount() - 1; { //DoCallCount is 1-indexed
						case responseNumber < timestampChangedIndex1:
							response = stubResponses[responseNumber]
						case responseNumber < timestampChangedIndex2:
							responseNumber = responseNumber - timestampChangedIndex1
							response = stubResponses[responseNumber]
						case responseNumber < timestampChangedIndex3:
							responseNumber = responseNumber - timestampChangedIndex2
							response = stubResponses[responseNumber]
						default:
							responseNumber = responseNumber - timestampChangedIndex3
							response = stubResponses[responseNumber]
						}

						jsonBytes, err := json.Marshal(response)
						Expect(err).NotTo(HaveOccurred())

						err = json.Unmarshal(jsonBytes, resp)
						Expect(err).NotTo(HaveOccurred())
						return nil
					}
				})

				It("return an error", func() {
					securityGroups, err := client.GetSecurityGroups(token)
					Expect(securityGroups).To(BeEmpty())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Ran out of retry attempts"))
					Expect(err.Error()).To(ContainSubstring("last_update time has changed"))
				})
			})
		})
	})
})

func stubDefaultLastUpdateRequest(fakeInternalJSONClient *fakes.JSONClient) {
	stubLastUpdateRequestWith(fakeInternalJSONClient, time.Now(), nil)
}

func stubLastUpdateRequestWith(fakeInternalJSONClient *fakes.JSONClient, timestamp time.Time, err error) {
	fakeInternalJSONClient.DoStub = func(method, route string, req, resp interface{}, token string) error {
		loadLastUpdateResponseIntoObject(timestamp, resp)
		return err
	}
}

func stubSecurityGroupRequestWith(fakeExternalJSONClient *fakes.JSONClient, securityGroups []cc_client.SecurityGroupResource, err error) {
	fakeExternalJSONClient.DoStub = func(method, route string, req, resp interface{}, token string) error {
		loadSecurityGroupsResponseIntoObject(securityGroups, resp)
		return err
	}
}

func loadLastUpdateResponseIntoObject(timestamp time.Time, response interface{}) {
	lastUpdateResponse := cc_client.SecurityGroupLatestUpdateResponse{
		LastUpdate: timestamp.Format(time.RFC3339),
	}

	jsonBytes, err := json.Marshal(lastUpdateResponse)
	Expect(err).NotTo(HaveOccurred())

	err = json.Unmarshal(jsonBytes, response)
	Expect(err).NotTo(HaveOccurred())
}

func loadSecurityGroupsResponseIntoObject(securityGroups []cc_client.SecurityGroupResource, response interface{}) {
	securityGroupsResponse := cc_client.GetSecurityGroupsResponse{
		Pagination: cc_client.Pagination{
			TotalPages: 1,
			Next:       cc_client.Href{Href: ""},
		},
		Resources: securityGroups,
	}

	jsonBytes, err := json.Marshal(securityGroupsResponse)
	Expect(err).NotTo(HaveOccurred())

	err = json.Unmarshal(jsonBytes, response)
	Expect(err).NotTo(HaveOccurred())
}
