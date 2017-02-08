package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"lib/testsupport"
	"math/rand"
	"net/http"
	"netmon/integration/fakes"
	"policy-server/config"
	"policy-server/models"
	"strings"
	"sync/atomic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("External API", func() {
	var (
		sessions          []*gexec.Session
		conf              config.Config
		policyServerConfs []config.Config
		testDatabase      *testsupport.TestDatabase

		fakeMetron fakes.FakeMetron
	)

	BeforeEach(func() {
		fakeMetron = fakes.New()

		dbName := fmt.Sprintf("test_netman_database_%x", rand.Int())
		dbConnectionInfo := testsupport.GetDBConnectionInfo()
		testDatabase = dbConnectionInfo.CreateDatabase(dbName)

		template := DefaultTestConfig(testDatabase.DBConfig(), fakeMetron.Address())
		policyServerConfs = configurePolicyServers(template, 2)
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]
	})

	AfterEach(func() {
		stopPolicyServers(sessions)

		if testDatabase != nil {
			testDatabase.Destroy()
		}

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Describe("authentication", func() {
		var makeNewRequest = func(method, route, bodyString string) *http.Request {
			var body io.Reader
			if bodyString != "" {
				body = strings.NewReader(bodyString)
			}
			url := fmt.Sprintf("http://%s:%d/%s", conf.ListenHost, conf.ListenPort, route)
			req, err := http.NewRequest(method, url, body)
			Expect(err).NotTo(HaveOccurred())

			return req
		}

		var TestMissingAuthHeader = func(req *http.Request) {
			By("check that 401 is returned when auth header is missing")
			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(responseString).To(MatchJSON(`{ "error": "missing authorization header"}`))
		}

		var TestBadBearerToken = func(req *http.Request) {
			By("check that 403 is returned when auth header is invalid")
			req.Header.Set("Authorization", "Bearer bad-token")

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(responseString).To(MatchJSON(`{ "error": "failed to verify token with uaa" }`))
		}

		var _ = DescribeTable("all the routes",
			func(method, route, bodyString string) {
				TestMissingAuthHeader(makeNewRequest(method, route, bodyString))
				TestBadBearerToken(makeNewRequest(method, route, bodyString))
			},
			Entry("POST to policies",
				"POST",
				"networking/v0/external/policies",
				`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`,
			),
			Entry("GET to policies",
				"GET",
				"networking/v0/external/policies",
				``,
			),
			Entry("POST to policies/delete",
				"POST",
				"networking/v0/external/policies/delete",
				`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`,
			),
		)
	})

	Describe("space developer", func() {
		var makeNewRequest = func(method, route, bodyString string) *http.Request {
			var body io.Reader
			if bodyString != "" {
				body = strings.NewReader(bodyString)
			}
			url := fmt.Sprintf("http://%s:%d/%s", conf.ListenHost, conf.ListenPort, route)
			req, err := http.NewRequest(method, url, body)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Set("Authorization", "Bearer space-dev-with-network-write-token")
			return req
		}

		Describe("Create policies", func() {
			var req *http.Request
			BeforeEach(func() {
				body := `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`
				req = makeNewRequest("POST", "networking/v0/external/policies", body)
			})
			It("succeeds for developers with access to apps and network.write permission", func() {
				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
					HaveName("ExternalPoliciesCreateRequestTime"),
				))
			})

			Context("when they do not have the network.write scope", func() {
				BeforeEach(func() {
					req.Header.Set("Authorization", "Bearer space-dev-token")
				})
				It("returns a 403 with a meaninful error", func() {
					resp, err := http.DefaultClient.Do(req)
					Expect(err).NotTo(HaveOccurred())

					Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
					responseString, err := ioutil.ReadAll(resp.Body)
					Expect(responseString).To(MatchJSON(`{ "error": "token missing allowed scopes: [network.admin network.write]"}`))

					Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
						HaveName("ExternalPoliciesCreateRequestTime"),
					))
				})
			})
			Context("when one app is in spaces they do not have access to", func() {
				BeforeEach(func() {
					body := `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "app-guid-not-in-my-spaces", "protocol": "tcp", "port": 8090 } } ] }`
					req = makeNewRequest("POST", "networking/v0/external/policies", body)
				})
				It("returns a 403 with a meaningful error", func() {
					resp, err := http.DefaultClient.Do(req)
					Expect(err).NotTo(HaveOccurred())

					Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
					responseString, err := ioutil.ReadAll(resp.Body)
					Expect(responseString).To(MatchJSON(`{ "error": "one or more applications cannot be found or accessed"}`))

					Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
						HaveName("ExternalPoliciesCreateRequestTime"),
					))
				})
			})

			It("fails for requests with bodies larger than 10 MB", func() {
				elevenMB := 11 << 20
				bytes := make([]byte, elevenMB, elevenMB)

				req := makeNewRequest("POST", "networking/v0/external/policies", string(bytes))
				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				responseString, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(MatchJSON(`{"error": "invalid request body"}`))

				Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
					HaveName("ExternalPoliciesCreateRequestTime"),
				))
			})
		})

		Describe("Delete policies", func() {
			var req *http.Request
			BeforeEach(func() {
				body := `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`
				req = makeNewRequest("POST", "networking/v0/external/policies/delete", body)
			})
			It("succeeds for developers with access to apps and network.write permission", func() {
				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})

			Context("when they do not have the network.write scope", func() {
				BeforeEach(func() {
					req.Header.Set("Authorization", "Bearer space-dev-token")
				})
				It("returns a 403 with a meaninful error", func() {
					resp, err := http.DefaultClient.Do(req)
					Expect(err).NotTo(HaveOccurred())

					Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
					responseString, err := ioutil.ReadAll(resp.Body)
					Expect(responseString).To(MatchJSON(`{ "error": "token missing allowed scopes: [network.admin network.write]"}`))
				})
			})
			Context("when one app is in spaces they do not have access to", func() {
				BeforeEach(func() {
					body := `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "app-guid-not-in-my-spaces", "protocol": "tcp", "port": 8090 } } ] }`
					req = makeNewRequest("POST", "networking/v0/external/policies/delete", body)
				})
				It("returns a 403 with a meaningful error", func() {
					resp, err := http.DefaultClient.Do(req)
					Expect(err).NotTo(HaveOccurred())

					Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
					responseString, err := ioutil.ReadAll(resp.Body)
					Expect(responseString).To(MatchJSON(`{ "error": "one or more applications cannot be found or accessed"}`))
				})
			})
		})

		Describe("List policies", func() {
			var req *http.Request
			BeforeEach(func() {
				req = makeNewRequest("GET", "networking/v0/external/policies", "")
			})

			Context("when there are no policies", func() {
				It("succeeds", func() {
					resp, err := http.DefaultClient.Do(req)
					Expect(err).NotTo(HaveOccurred())

					Expect(resp.StatusCode).To(Equal(http.StatusOK))
					responseString, err := ioutil.ReadAll(resp.Body)
					Expect(responseString).To(MatchJSON(`{
					"total_policies": 0,
					"policies": []
				}`))
				})
			})

			Context("when there are policies in spaces the user does not belong to", func() {
				BeforeEach(func() {
					body := `{ "policies": [ {"source": { "id": "live-app-1-guid" }, "destination": { "id": "live-app-5-guid", "protocol": "tcp", "port": 8090 } } ] }`
					req = makeNewRequest("POST", "networking/v0/external/policies", body)
					req.Header.Set("Authorization", "Bearer valid-token")
					_, err := http.DefaultClient.Do(req)
					Expect(err).NotTo(HaveOccurred())

					body = `{ "policies": [ {"source": { "id": "live-app-1-guid" }, "destination": { "id": "live-app-2-guid", "protocol": "tcp", "port": 8090 } } ] }`
					req = makeNewRequest("POST", "networking/v0/external/policies", body)
					req.Header.Set("Authorization", "Bearer valid-token")
					_, err = http.DefaultClient.Do(req)
					Expect(err).NotTo(HaveOccurred())
				})
				It("does not return those policies", func() {
					req = makeNewRequest("GET", "networking/v0/external/policies", "")
					resp, err := http.DefaultClient.Do(req)
					Expect(err).NotTo(HaveOccurred())

					Expect(resp.StatusCode).To(Equal(http.StatusOK))
					responseString, err := ioutil.ReadAll(resp.Body)
					expectedResp := `{
						"total_policies": 1,
						"policies": [ {"source": { "id": "live-app-1-guid" }, "destination": { "id": "live-app-2-guid", "protocol": "tcp", "port": 8090 } } ] 
					}`
					Expect(responseString).To(MatchJSON(expectedResp))
				})
			})

			Context("when they do not have the network.write scope", func() {
				BeforeEach(func() {
					req.Header.Set("Authorization", "Bearer space-dev-token")
				})
				It("returns a 403 with a meaningful error", func() {
					resp, err := http.DefaultClient.Do(req)
					Expect(err).NotTo(HaveOccurred())

					Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
					responseString, err := ioutil.ReadAll(resp.Body)
					Expect(responseString).To(MatchJSON(`{ "error": "token missing allowed scopes: [network.admin network.write]"}`))
				})
			})
		})
	})

	Context("when there are concurrent create requests", func() {
		It("remains consistent", func() {
			policiesRoute := "external/policies"
			add := func(policy models.Policy) {
				requestBody, _ := json.Marshal(map[string]interface{}{
					"policies": []models.Policy{policy},
				})
				resp := makeAndDoRequest("POST", policyServerUrl(policiesRoute, policyServerConfs), bytes.NewReader(requestBody))
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(MatchJSON("{}"))
			}

			nPolicies := 100
			policies := []interface{}{}
			for i := 0; i < nPolicies; i++ {
				appName := fmt.Sprintf("some-app-%x", i)
				policies = append(policies, models.Policy{
					Source:      models.Source{ID: appName},
					Destination: models.Destination{ID: appName, Protocol: "tcp", Port: 1234},
				})
			}

			parallelRunner := &testsupport.ParallelRunner{
				NumWorkers: 4,
			}
			By("adding lots of policies concurrently")
			var nAdded int32
			parallelRunner.RunOnSlice(policies, func(policy interface{}) {
				add(policy.(models.Policy))
				atomic.AddInt32(&nAdded, 1)
			})
			Expect(nAdded).To(Equal(int32(nPolicies)))

			By("getting all the policies")
			resp := makeAndDoRequest("GET", policyServerUrl(policiesRoute, policyServerConfs), nil)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseBytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			var policiesResponse struct {
				TotalPolicies int             `json:"total_policies"`
				Policies      []models.Policy `json:"policies"`
			}
			Expect(json.Unmarshal(responseBytes, &policiesResponse)).To(Succeed())

			Expect(policiesResponse.TotalPolicies).To(Equal(nPolicies))

			By("verifying all the policies are present")
			for _, policy := range policies {
				Expect(policiesResponse.Policies).To(ContainElement(policy))
			}

			By("verify tags")
			tagsRoute := "external/tags"
			resp = makeAndDoRequest("GET", policyServerUrl(tagsRoute, policyServerConfs), nil)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseBytes, err = ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			var tagsResponse struct {
				Tags []models.Tag `json:"tags"`
			}
			Expect(json.Unmarshal(responseBytes, &tagsResponse)).To(Succeed())
			Expect(tagsResponse.Tags).To(HaveLen(nPolicies))
		})
	})

	Context("when these are concurrent create and delete requests", func() {
		It("remains consistent", func() {
			baseUrl := fmt.Sprintf("http://%s:%d", conf.ListenHost, conf.ListenPort)
			policiesUrl := fmt.Sprintf("%s/networking/v0/external/policies", baseUrl)
			policiesDeleteUrl := fmt.Sprintf("%s/networking/v0/external/policies/delete", baseUrl)

			do := func(method, url string, policy models.Policy) {
				requestBody, _ := json.Marshal(map[string]interface{}{
					"policies": []models.Policy{policy},
				})
				resp := makeAndDoRequest(method, url, bytes.NewReader(requestBody))
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(MatchJSON("{}"))
			}

			nPolicies := 100
			policies := []interface{}{}
			for i := 0; i < nPolicies; i++ {
				appName := fmt.Sprintf("some-app-%x", i)
				policies = append(policies, models.Policy{
					Source:      models.Source{ID: appName},
					Destination: models.Destination{ID: appName, Protocol: "tcp", Port: 1234},
				})
			}

			parallelRunner := &testsupport.ParallelRunner{
				NumWorkers: 4,
			}
			toDelete := make(chan (interface{}), nPolicies)

			go func() {
				parallelRunner.RunOnSlice(policies, func(policy interface{}) {
					p := policy.(models.Policy)
					do("POST", policiesUrl, p)
					toDelete <- p
				})
				close(toDelete)
			}()

			var nDeleted int32
			parallelRunner.RunOnChannel(toDelete, func(policy interface{}) {
				p := policy.(models.Policy)
				do("POST", policiesDeleteUrl, p)
				atomic.AddInt32(&nDeleted, 1)
			})

			Expect(nDeleted).To(Equal(int32(nPolicies)))

			resp := makeAndDoRequest("GET", policiesUrl, nil)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseBytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			var policiesResponse struct {
				TotalPolicies int             `json:"total_policies"`
				Policies      []models.Policy `json:"policies"`
			}
			Expect(json.Unmarshal(responseBytes, &policiesResponse)).To(Succeed())

			Expect(policiesResponse.TotalPolicies).To(Equal(0))
		})
	})

	Describe("adding policies", func() {
		It("responds with 200 and a body of {} and we can see it in the list", func() {
			body := strings.NewReader(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`)
			resp := makeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
				body,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(MatchJSON("{}"))

			resp = makeAndDoRequest(
				"GET",
				fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
				nil,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err = ioutil.ReadAll(resp.Body)
			Expect(responseString).To(MatchJSON(`{
				"total_policies": 1,
				"policies": [
					{ "source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } }
				]}`))
		})

		Context("when the protocol is invalid", func() {
			It("gives a helpful error", func() {
				body := strings.NewReader(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "nope", "port": 8090 } } ] }`)
				resp := makeAndDoRequest(
					"POST",
					fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
					body,
				)

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				responseString, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(MatchJSON(`{ "error": "invalid destination protocol, specify either udp or tcp" }`))
			})
		})
		Context("when the port is invalid", func() {
			It("gives a helpful error", func() {
				body := strings.NewReader(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 0 } } ] }`)
				resp := makeAndDoRequest(
					"POST",
					fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
					body,
				)

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				responseString, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(MatchJSON(`{ "error": "invalid destination port value 0, must be 1-65535" }`))
			})
		})
	})

	Describe("cleanup policies", func() {
		BeforeEach(func() {
			body := strings.NewReader(`{ "policies": [
				{"source": { "id": "live-app-1-guid" }, "destination": { "id": "live-app-2-guid", "protocol": "tcp", "port": 8080 } },
				{"source": { "id": "live-app-2-guid" }, "destination": { "id": "live-app-2-guid", "protocol": "tcp", "port": 9999 } },
				{"source": { "id": "live-app-1-guid" }, "destination": { "id": "dead-app", "protocol": "tcp", "port": 3333 } }
				]} `)

			resp := makeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
				body,
			)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

		})

		It("responds with a 200 and lists all stale policies", func() {
			resp := makeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/v0/external/policies/cleanup", conf.ListenHost, conf.ListenPort),
				nil,
			)

			stalePoliciesStr := `{
				"total_policies":1,
				"policies": [
				 {"source": { "id": "live-app-1-guid" }, "destination": { "id": "dead-app", "protocol": "tcp", "port": 3333 } }
				 ]}
				`

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			Expect(bodyBytes).To(MatchJSON(stalePoliciesStr))
			Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
				HaveName("ExternalPoliciesCleanupRequestTime"),
			))
		})
	})

	Describe("listing policies", func() {
		Context("when providing a list of ids as a query parameter", func() {
			It("responds with a 200 and lists all policies which contain one of those ids", func() {
				resp := makeAndDoRequest(
					"POST",
					fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
					strings.NewReader(`{ "policies": [
						{"source": { "id": "app1" }, "destination": { "id": "app2", "protocol": "tcp", "port": 8080 } },
						{"source": { "id": "app3" }, "destination": { "id": "app1", "protocol": "tcp", "port": 9999 } },
						{"source": { "id": "app3" }, "destination": { "id": "app4", "protocol": "tcp", "port": 3333 } }
					]}
					`),
				)

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(MatchJSON("{}"))

				resp = makeAndDoRequest(
					"GET",
					fmt.Sprintf("http://%s:%d/networking/v0/external/policies?id=app1,app2", conf.ListenHost, conf.ListenPort),
					nil,
				)

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err = ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(MatchJSON(`{
					"total_policies": 2,	
					"policies": [
				 {"source": { "id": "app1" }, "destination": { "id": "app2", "protocol": "tcp", "port": 8080 } },
				 {"source": { "id": "app3" }, "destination": { "id": "app1", "protocol": "tcp", "port": 9999 } }
				 ]}
				`))

				By("emitting metrics about durations")
				Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
					HaveName("ExternalPoliciesIndexRequestTime"),
				))
			})
		})
	})

	Describe("deleting policies", func() {
		BeforeEach(func() {
			resp := makeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
				strings.NewReader(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`),
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(MatchJSON("{}"))
		})

		Context("when all of the deletes succeed", func() {
			It("responds with 200 and a body of {} and we can see it is removed from the list", func() {

				body := strings.NewReader(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`)

				response := makeAndDoRequest(
					"POST",
					fmt.Sprintf("http://%s:%d/networking/v0/external/policies/delete", conf.ListenHost, conf.ListenPort),
					body,
				)

				Expect(response.StatusCode).To(Equal(http.StatusOK))
				responseString, err := ioutil.ReadAll(response.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(MatchJSON(`{}`))

				response = makeAndDoRequest(
					"GET",
					fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
					nil,
				)

				Expect(response.StatusCode).To(Equal(http.StatusOK))
				responseString, err = ioutil.ReadAll(response.Body)
				Expect(responseString).To(MatchJSON(`{
					"total_policies": 0,
					"policies": []
				}`))

				By("emitting metrics about durations")
				Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
					HaveName("ExternalPoliciesDeleteRequestTime"),
				))
			})

		})

		Context("when one of the policies to delete does not exist", func() {
			It("responds with status 200", func() {
				body := strings.NewReader(`{ "policies": [
						{"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } },
						{"source": { "id": "some-non-existent-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } }
					] }`)

				response := makeAndDoRequest(
					"POST",
					fmt.Sprintf("http://%s:%d/networking/v0/external/policies/delete", conf.ListenHost, conf.ListenPort),
					body,
				)

				Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
					HaveName("ExternalPoliciesDeleteRequestTime"),
				))

				Expect(response.StatusCode).To(Equal(http.StatusOK))
				responseString, err := ioutil.ReadAll(response.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(MatchJSON(`{}`))

				By("emitting metrics about durations")
				Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
					HaveName("ExternalPoliciesDeleteRequestTime"),
				))
			})
		})
	})

	Describe("listing tags", func() {
		BeforeEach(func() {
			body := strings.NewReader(`{ "policies": [
			{"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } },
			{"source": { "id": "some-app-guid" }, "destination": { "id": "another-app-guid", "protocol": "udp", "port": 6666 } },
			{"source": { "id": "another-app-guid" }, "destination": { "id": "some-app-guid", "protocol": "tcp", "port": 3333 } }
			] }`)
			resp := makeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
				body,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(MatchJSON("{}"))
		})

		It("returns a list of application guid to tag mapping", func() {
			By("listing the current tags")
			resp := makeAndDoRequest(
				"GET",
				fmt.Sprintf("http://%s:%d/networking/v0/external/tags", conf.ListenHost, conf.ListenPort),
				nil,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(MatchJSON(`{ "tags": [
				{ "id": "some-app-guid", "tag": "01" },
				{ "id": "some-other-app-guid", "tag": "02" },
				{ "id": "another-app-guid", "tag": "03" }
			] }`))

			By("reusing tags that are no longer in use")
			body := strings.NewReader(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`)
			resp = makeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/v0/external/policies/delete", conf.ListenHost, conf.ListenPort),
				body,
			)

			body = strings.NewReader(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "yet-another-app-guid", "protocol": "udp", "port": 4567 } } ] }`)
			resp = makeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
				body,
			)

			resp = makeAndDoRequest(
				"GET",
				fmt.Sprintf("http://%s:%d/networking/v0/external/tags", conf.ListenHost, conf.ListenPort),
				nil,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err = ioutil.ReadAll(resp.Body)
			Expect(responseString).To(MatchJSON(`{ "tags": [
				{ "id": "some-app-guid", "tag": "01" },
				{ "id": "yet-another-app-guid", "tag": "02" },
				{ "id": "another-app-guid", "tag": "03" }
			] }`))

			Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
				HaveName("ExternalPoliciesTagsIndexRequestTime"),
			))
		})
	})
})
