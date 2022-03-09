package cc_client_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/fakes"
	"code.cloudfoundry.org/cf-networking-helpers/json_client"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/policy-server/cc_client"
	. "code.cloudfoundry.org/policy-server/cc_client"
	"code.cloudfoundry.org/policy-server/cc_client/fixtures"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		client         *Client
		fakeJSONClient *fakes.JSONClient
		logger         *lagertest.TestLogger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		fakeJSONClient = &fakes.JSONClient{}
		client = &Client{
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
					"live-app-1-guid": {},
					"live-app-2-guid": {},
					"live-app-3-guid": {},
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
				"live-app-1-guid": {},
				"live-app-2-guid": {},
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

	Describe("GetLiveSpaceGUIDs", func() {
		var (
			passedToken string
			passedRoute string
		)

		BeforeEach(func() {
			fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
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
				fakeJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns the error", func() {
				_, err := client.GetLiveSpaceGUIDs("some-token", []string{})
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})

		Context("when there are multiple pages", func() {
			BeforeEach(func() {
				fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
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
			space := SpaceResponse{
				Entity: SpaceEntity{
					Name:             "name-2064",
					OrganizationGUID: "6e1ca5aa-55f1-4110-a97f-1f3473e771b9",
				},
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
				fakeJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns a helpful error", func() {
				_, err := client.GetAppSpaces("some-token", []string{"some-guid"})
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
			fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
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

			Expect(fakeJSONClient.DoCallCount()).To(Equal(3))

			method, route, reqData, _, token := fakeJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v2/users/some-subject-id/spaces?results-per-page=100"))
			Expect(reqData).To(BeNil())
			Expect(token).To(Equal("bearer some-token"))

			method, route, reqData, _, token = fakeJSONClient.DoArgsForCall(1)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v2/users/some-subject-id/spaces?order-direction=asc&page=2&results-per-page=1"))
			Expect(reqData).To(BeNil())
			Expect(token).To(Equal("bearer some-token"))

			method, route, reqData, _, token = fakeJSONClient.DoArgsForCall(2)

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
				fakeJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns a helpful error", func() {
				_, err := client.GetSubjectSpaces("some-token", "some-subject-id")
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})
	})

	Describe("GetSubjectSpace", func() {
		space := SpaceResponse{
			Entity: SpaceEntity{
				Name:             "some-space-name",
				OrganizationGUID: "some-org-guid",
			},
		}
		BeforeEach(func() {
			fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				_ = json.Unmarshal([]byte(fixtures.SubjectSpace), respData)
				return nil
			}
		})

		It("returns the matching spaces for the subject", func() {
			matchingSpace, err := client.GetSubjectSpace("some-token", "some-subject-id", space)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeJSONClient.DoCallCount()).To(Equal(1))

			method, route, reqData, _, token := fakeJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v2/spaces?q=developer_guid%3Asome-subject-id&q=name%3Asome-space-name&q=organization_guid%3Asome-org-guid"))
			Expect(reqData).To(BeNil())
			Expect(token).To(Equal("bearer some-token"))

			Expect(matchingSpace.Entity).To(Equal(space.Entity))
		})

		Context("when the subject has no spaces", func() {
			BeforeEach(func() {
				fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
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
				fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
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
				fakeJSONClient.DoReturns(errors.New("banana"))
			})

			It("returns a helpful error", func() {
				_, err := client.GetSubjectSpace("some-token", "some-subject-id", space)
				Expect(err).To(MatchError(ContainSubstring("json client do: banana")))
			})
		})
	})

	Describe("GetSecurityGroups", func() {
		BeforeEach(func() {
			fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
				err := json.Unmarshal([]byte(fixtures.OneSecurityGroup), respData)
				Expect(err).ToNot(HaveOccurred())
				return nil
			}
		})

		It("polls the Cloud Controller successfully", func() {
			sgs, err := client.GetSecurityGroups("some-token")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeJSONClient.DoCallCount()).To(Equal(1))

			method, route, reqData, _, _ := fakeJSONClient.DoArgsForCall(0)

			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/v3/security_groups?per_page=5000&order_by=created_at&page=1"))
			Expect(reqData).To(BeNil())

			Expect(len(sgs)).To(Equal(1))
			Expect(sgs[0].Name).To(Equal("my-group0"))
			Expect(sgs[0].GUID).To(Equal("b85a788e-671f-4549-814d-e34cdb2f539a"))
			Expect(sgs[0].GloballyEnabled.Running).To(BeTrue())
			Expect(sgs[0].GloballyEnabled.Staging).To(BeFalse())
			Expect(sgs[0].Rules[0].Protocol).To(Equal("tcp"))
			Expect(sgs[0].Rules[1].Protocol).To(Equal("icmp"))
			Expect(sgs[0].Relationships.StagingSpaces.Data).To(Equal([]map[string]string{
				{"guid": "space-guid-1"},
				{"guid": "space-guid-2"},
			}))
			Expect(sgs[0].Relationships.RunningSpaces.Data).To(Equal([]map[string]string{
				{"guid": "space-guid-3"},
				{"guid": "space-guid-4"},
			}))
		})

		Context("when there are no security groups", func() {
			BeforeEach(func() {
				fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					err := json.Unmarshal([]byte(fixtures.NoSecurityGroups), respData)
					Expect(err).ToNot(HaveOccurred())
					return nil
				}
			})

			It("Returns an empty set", func() {
				sgs, err := client.GetSecurityGroups("some-token")
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeJSONClient.DoCallCount()).To(Equal(1))

				method, route, reqData, _, _ := fakeJSONClient.DoArgsForCall(0)

				Expect(method).To(Equal("GET"))
				Expect(route).To(Equal("/v3/security_groups?per_page=5000&order_by=created_at&page=1"))
				Expect(reqData).To(BeNil())

				Expect(len(sgs)).To(Equal(0))
			})
		})

		Context("when multiple security groups are returned", func() {
			BeforeEach(func() {
				fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					err := json.Unmarshal([]byte(fixtures.TwoSecurityGroups), respData)
					Expect(err).ToNot(HaveOccurred())
					return nil
				}
			})

			It("Returns them all", func() {
				sgs, err := client.GetSecurityGroups("some-token")
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeJSONClient.DoCallCount()).To(Equal(1))

				method, route, reqData, _, _ := fakeJSONClient.DoArgsForCall(0)

				Expect(method).To(Equal("GET"))
				Expect(route).To(Equal("/v3/security_groups?per_page=5000&order_by=created_at&page=1"))
				Expect(reqData).To(BeNil())

				Expect(len(sgs)).To(Equal(2))
				Expect(sgs[1].Name).To(Equal("my-group2"))
				Expect(sgs[1].GUID).To(Equal("second-guid"))
				Expect(sgs[1].GloballyEnabled.Running).To(BeFalse())
				Expect(sgs[1].GloballyEnabled.Staging).To(BeTrue())
				Expect(sgs[1].Rules[0].Protocol).To(Equal("tcp"))
				Expect(sgs[1].Rules[0].Ports).To(Equal("53"))
			})
		})

		Context("when there are multiple pages", func() {
			var sgs []cc_client.SecurityGroupResource
			var pages int
			var deleteBeforePage int

			BeforeEach(func() {
				pages = 10
				deleteBeforePage = 0
				sgs = []cc_client.SecurityGroupResource{}
				// build a list of SGs that we can paginate over
				for i := 0; i < pages*cc_client.SecurityGroupsPerPage; i++ {
					sgs = append(sgs, cc_client.SecurityGroupResource{
						GUID: fmt.Sprintf("guid-%d", i),
					})
				}
			})
			JustBeforeEach(func() {
				// set up a fake client that will return the list of SGs in a paginated fashion
				// using consistent ordering, and adapting to changes in the per_page number + page request
				fakeJSONClient.DoStub = func(method, route string, reqData, respData interface{}, token string) error {
					url, err := url.Parse(route)
					Expect(err).ToNot(HaveOccurred())

					//grab per_page + page query params, or set sensible defaults
					perPageStr := url.Query().Get("per_page")
					if perPageStr == "" {
						perPageStr = "50"
					}
					pageStr := url.Query().Get("page")
					if pageStr == "" {
						pageStr = "1"
					}
					perPage, err := strconv.Atoi(perPageStr)
					Expect(err).ToNot(HaveOccurred())
					page, err := strconv.Atoi(pageStr)
					Expect(err).ToNot(HaveOccurred())
					next := page + 1

					// delete a number of indexes from our dataset, simulating deletions between
					// pagination requests. any number of deletions should trigger an error, so
					// delete the same number of indexes as our page is at
					if deleteBeforePage == page {
						sgs = sgs[page:]
					}

					resp := cc_client.GetSecurityGroupsResponse{}
					resp.Pagination.TotalPages = totalPagesForSet(perPage, len(sgs))
					//only set the next href if there's another page
					if next <= resp.Pagination.TotalPages {
						resp.Pagination.Next.Href = fmt.Sprintf("/v3/security_groups?per_page=%d&order_by=created_at&page=%d", perPage, next)
					}

					// figure out which results we should return
					first := (page - 1) * perPage
					last := first + perPage

					// avoid index errors on the last page of sg results
					if last > len(sgs) {
						last = len(sgs)
					}
					resp.Resources = sgs[first:last]

					// update the inbound respData with our new response
					v := reflect.ValueOf(respData)
					v.Elem().Set(reflect.ValueOf(resp))

					return nil
				}
			})
			It("queries with decreasing page sizes and increasing offsets to detect changes", func() {
				_, err := client.GetSecurityGroups("some-token")
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeJSONClient.DoCallCount()).To(Equal(totalPagesForSet(cc_client.SecurityGroupsPerPage-pages+1, len(sgs))))
				for i := 1; i <= pages; i++ {
					method, route, reqData, _, _ := fakeJSONClient.DoArgsForCall(i - 1)
					Expect(method).To(Equal("GET"))
					Expect(route).To(Equal(fmt.Sprintf("/v3/security_groups?per_page=%d&order_by=created_at&page=%d", cc_client.SecurityGroupsPerPage-i+1, i)))
					Expect(reqData).To(BeNil())
				}
			})

			It("returns all the security groups", func() {
				returnedSGs, err := client.GetSecurityGroups("some-token")
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeJSONClient.DoCallCount()).To(Equal(11))

				Expect(returnedSGs).To(Equal(sgs))
			})

			Context("when entries are removed from previous pages", func() {
				// skip page 1 since it would never trigger an error
				It("detects a changes and invalidates the by throwing an error", func() {
					for page := 2; page <= pages; page++ {
						deleteBeforePage = page
						_, err := client.GetSecurityGroups("some-token")
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError(cc_client.NewUnstableSecurityGroupListError(fmt.Errorf("unexpected SG changes during pagination"))))
						_, route, _, _, _ := fakeJSONClient.DoArgsForCall(fakeJSONClient.DoCallCount() - 1)
						Expect(strings.Split(route, "?")[1]).To(Equal(fmt.Sprintf("per_page=%d&order_by=created_at&page=%d", cc_client.SecurityGroupsPerPage-page+1, page)))
					}
				})
			})
		})

		Context("when the json client returns an error", func() {
			BeforeEach(func() {
				fakeJSONClient.DoReturns(errors.New("kissa ja undulaatti"))
			})

			It("returns a helpful error", func() {
				_, err := client.GetSubjectSpaces("some-token", "some-subject-id")
				Expect(err).To(MatchError(ContainSubstring("json client do: kissa ja undulaatti")))
			})
		})
	})
})

func totalPagesForSet(perPage, setLength int) int {
	return int(math.Ceil(float64(setLength) / float64(perPage)))
}
