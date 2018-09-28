package integration_test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"policy-server/api"
	"policy-server/config"
	"policy-server/integration/helpers"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("External API Space Developer", func() {
	var (
		sessions          []*gexec.Session
		conf              config.Config
		policyServerConfs []config.Config
		dbConf            db.Config

		fakeMetron metrics.FakeMetron
	)

	BeforeEach(func() {
		fakeMetron = metrics.NewFakeMetron()

		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("space_developer_test_node_%d", ports.PickAPort())

		template, _ := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
		policyServerConfs = configurePolicyServers(template, 2)
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]
	})

	AfterEach(func() {
		stopPolicyServers(sessions, policyServerConfs)
		Expect(fakeMetron.Close()).To(Succeed())
	})

	Describe("space developer", func() {
		makeNewRequest := func(method, route, bodyString string) *http.Request {
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
			var (
				req  *http.Request
				body string
			)
			BeforeEach(func() {
				body = `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "ports": { "start": 8090, "end": 8090 } } } ] }`
				req = makeNewRequest("POST", "networking/v1/external/policies", body)
			})

			Context("when space developer self-service is disabled", func() {
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
						Expect(responseString).To(MatchJSON(`{ "error": "provided scopes [] do not include allowed scopes [network.admin network.write]"}`))
					})
				})

				Context("when one app is in spaces they do not have access to", func() {
					BeforeEach(func() {
						body = `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "app-guid-not-in-my-spaces", "protocol": "tcp", "ports": { "start": 8090, "end": 8090 } } } ] }`
						req = makeNewRequest("POST", "networking/v1/external/policies", body)
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

			Context("when space developer self-service is enabled", func() {
				BeforeEach(func() {
					stopPolicyServers(sessions, policyServerConfs)

					template, _ := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
					template.EnableSpaceDeveloperSelfService = true
					policyServerConfs = configurePolicyServers(template, 2)

					//if run in parallel sessions could update/override sessions in a different goroutine
					sessions = startPolicyServers(policyServerConfs)
					conf = policyServerConfs[0]

					req = makeNewRequest("POST", "networking/v1/external/policies", body)
					req.Header.Set("Authorization", "Bearer space-dev-token")
				})

				It("succeeds for developers with access to apps", func() {
					resp, err := http.DefaultClient.Do(req)
					Expect(err).NotTo(HaveOccurred())

					Expect(resp.StatusCode).To(Equal(http.StatusOK))
				})

				Context("when one app is in spaces they do not have access to", func() {
					BeforeEach(func() {
						body = `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "app-guid-not-in-my-spaces", "protocol": "tcp", "ports": { "start": 8090, "end": 8090 } } } ] }`
						req = makeNewRequest("POST", "networking/v1/external/policies", body)
						req.Header.Set("Authorization", "Bearer space-dev-token")
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

			It("fails for requests with bodies larger than 10 MB", func() {
				elevenMB := 11 << 20
				bytes := make([]byte, elevenMB, elevenMB)

				req := makeNewRequest("POST", "networking/v1/external/policies", string(bytes))
				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				responseString, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(MatchJSON(`{"error": "failed reading request body"}`))
			})
		})

		Describe("Quotas", func() {
			var (
				req  *http.Request
				body string
			)

			BeforeEach(func() {
				body = `{ "policies": [
				{"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "ports": { "start": 8090, "end": 8090 } } },
				{"source": { "id": "some-app-guid" }, "destination": { "id": "another-app-guid", "protocol": "udp", "ports": { "start": 7070, "end": 7070 } } }
				] }`
				req = makeNewRequest("POST", "networking/v1/external/policies", body)
			})
			It("rejects requests to add policies above the quota", func() {
				By("adding the maximum allowed policies")
				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				By("seeing that adding another policy fails")
				body = `{ "policies": [
				{"source": { "id": "some-app-guid" }, "destination": { "id": "yet-another-other-app-guid", "protocol": "tcp", "ports": { "start": 9000, "end": 9000 } } }
				] }`
				req = makeNewRequest("POST", "networking/v1/external/policies", body)
				resp, err = http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
				responseString, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(MatchJSON(`{"error": "policy quota exceeded"}`))

				By("deleting a policy")
				body = `{ "policies": [
				{"source": { "id": "some-app-guid" }, "destination": { "id": "another-app-guid", "protocol": "udp", "ports": { "start": 7070, "end": 7070 } } }
				] }`
				req = makeNewRequest("POST", "networking/v1/external/policies/delete", body)
				resp, err = http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				By("seeing that adding another policy succeeds")
				body = `{ "policies": [
				{"source": { "id": "some-app-guid" }, "destination": { "id": "yet-another-other-app-guid", "protocol": "tcp", "ports": { "start": 9000, "end": 9000 } } }
				] }`
				req = makeNewRequest("POST", "networking/v1/external/policies", body)
				resp, err = http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
		})

		Describe("Delete policies", func() {
			var req *http.Request
			BeforeEach(func() {
				body := `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "ports": { "start": 8090, "end": 8090 } } } ] }`
				req = makeNewRequest("POST", "networking/v1/external/policies/delete", body)
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
					Expect(responseString).To(MatchJSON(`{ "error": "provided scopes [] do not include allowed scopes [network.admin network.write]"}`))
				})
			})
			Context("when one app is in spaces they do not have access to", func() {
				BeforeEach(func() {
					body := `{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "app-guid-not-in-my-spaces", "protocol": "tcp", "ports": { "start": 8090, "end": 8090 } } } ] }`
					req = makeNewRequest("POST", "networking/v1/external/policies/delete", body)
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
				req = makeNewRequest("GET", "networking/v1/external/policies", "")
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
					policies := []api.Policy{}
					for i := 0; i < 150; i++ {
						policies = append(policies, api.Policy{
							Source: api.Source{ID: "live-app-1-guid"},
							Destination: api.Destination{ID: fmt.Sprintf("not-in-space-app-%d-guid", i),
								Ports: api.Ports{
									Start: 8090,
									End:   8090,
								},
								Protocol: "tcp",
							},
						})
					}
					policies = append(policies, api.Policy{
						Source: api.Source{ID: "live-app-1-guid"},
						Destination: api.Destination{ID: "live-app-2-guid",
							Ports: api.Ports{
								Start: 8090,
								End:   8090,
							},
							Protocol: "tcp",
						},
					})

					body := map[string][]api.Policy{
						"policies": policies,
					}
					bodyBytes, err := json.Marshal(body)
					Expect(err).NotTo(HaveOccurred())

					req = makeNewRequest("POST", "networking/v1/external/policies", string(bodyBytes))
					req.Header.Set("Authorization", "Bearer valid-token")
					_, err = http.DefaultClient.Do(req)
					Expect(err).NotTo(HaveOccurred())
				})

				It("does not return those policies", func() {
					req = makeNewRequest("GET", "networking/v1/external/policies", "")
					resp, err := http.DefaultClient.Do(req)
					Expect(err).NotTo(HaveOccurred())

					Expect(resp.StatusCode).To(Equal(http.StatusOK))
					responseString, err := ioutil.ReadAll(resp.Body)
					expectedResp := `{
						"total_policies": 1,
						"policies": [ {"source": { "id": "live-app-1-guid" }, "destination": { "id": "live-app-2-guid", "protocol": "tcp", "ports": { "start": 8090, "end": 8090 }}} ]
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
					Expect(responseString).To(MatchJSON(`{ "error": "provided scopes [] do not include allowed scopes [network.admin network.write]"}`))
				})
			})
		})

		Describe("Egress Policy and Destination Endpoints", func() {
			It("does not allow access", func() {
				req := makeNewRequest("GET", "networking/v1/external/destinations", "")
				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

				req = makeNewRequest("POST", "networking/v1/external/destinations", "{}")
				resp, err = http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

				req = makeNewRequest("DELETE", "networking/v1/external/destinations/meow", "")
				resp, err = http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

				req = makeNewRequest("POST", "networking/v1/external/egress_policies", "{}")
				resp, err = http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

				req = makeNewRequest("DELETE", "networking/v1/external/egress_policies/meow", "")
				resp, err = http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
			})
		})
	})
})
