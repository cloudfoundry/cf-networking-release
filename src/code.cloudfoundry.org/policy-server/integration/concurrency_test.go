package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	"code.cloudfoundry.org/policy-server/api"
	"code.cloudfoundry.org/policy-server/config"
	"code.cloudfoundry.org/policy-server/integration/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("External API Concurrency", func() {
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
		dbConf.DatabaseName = fmt.Sprintf("concurrency_test_node_%d", ports.PickAPort())

		template, _, _ := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
		policyServerConfs = configurePolicyServers(template, 2)
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]
	})

	AfterEach(func() {
		stopPolicyServers(sessions, policyServerConfs)

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Context("when there are concurrent create requests", func() {
		It("remains consistent", func() {
			policiesRoute := "external/policies"
			add := func(policy api.Policy) {
				requestBody, _ := json.Marshal(map[string]interface{}{
					"policies": []api.Policy{policy},
				})
				resp := helpers.MakeAndDoRequest("POST", policyServerUrl(policiesRoute, policyServerConfs), nil, bytes.NewReader(requestBody))
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(MatchJSON("{}"))
			}

			nPolicies := 100
			policies := []interface{}{}
			for i := 0; i < nPolicies; i++ {
				appName := fmt.Sprintf("some-app-%x", i)
				policies = append(policies, api.Policy{
					Source: api.Source{ID: appName},
					Destination: api.Destination{
						ID:       appName,
						Protocol: "tcp",
						Ports: api.Ports{
							Start: 1234,
							End:   1234,
						},
					},
				})
			}

			parallelRunner := &testsupport.ParallelRunner{
				NumWorkers: 4,
			}
			By("adding lots of policies concurrently")
			var nAdded int32
			parallelRunner.RunOnSlice(policies, func(policy interface{}) {
				add(policy.(api.Policy))
				atomic.AddInt32(&nAdded, 1)
			})
			Expect(nAdded).To(Equal(int32(nPolicies)))

			By("getting all the policies")
			resp := helpers.MakeAndDoRequest("GET", policyServerUrl(policiesRoute, policyServerConfs), nil, nil)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseBytes, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			var policiesResponse struct {
				TotalPolicies int          `json:"total_policies"`
				Policies      []api.Policy `json:"policies"`
			}
			Expect(json.Unmarshal(responseBytes, &policiesResponse)).To(Succeed())

			Expect(policiesResponse.TotalPolicies).To(Equal(nPolicies))

			By("verifying all the policies are present")
			for _, policy := range policies {
				Expect(policiesResponse.Policies).To(ContainElement(policy))
			}

			By("verify tags")
			tagsRoute := "external/tags"
			resp = helpers.MakeAndDoRequest("GET", policyServerUrl(tagsRoute, policyServerConfs), nil, nil)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseBytes, err = io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			var tagsResponse struct {
				Tags []api.Tag `json:"tags"`
			}
			Expect(json.Unmarshal(responseBytes, &tagsResponse)).To(Succeed())
			Expect(tagsResponse.Tags).To(HaveLen(nPolicies))
		})
	})

	Context("when these are concurrent create and delete requests", func() {
		It("remains consistent", func() {
			baseUrl := fmt.Sprintf("http://%s:%d", conf.ListenHost, conf.ListenPort)
			policiesUrl := fmt.Sprintf("%s/networking/v1/external/policies", baseUrl)
			policiesDeleteUrl := fmt.Sprintf("%s/networking/v1/external/policies/delete", baseUrl)

			do := func(method, url string, policy api.Policy) {
				requestBody, _ := json.Marshal(map[string]interface{}{
					"policies": []api.Policy{policy},
				})
				resp := helpers.MakeAndDoRequest(method, url, nil, bytes.NewReader(requestBody))
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(MatchJSON("{}"))
			}

			nPolicies := 100
			policies := []interface{}{}
			for i := 0; i < nPolicies; i++ {
				appName := fmt.Sprintf("some-app-%x", i)
				policies = append(policies, api.Policy{
					Source: api.Source{ID: appName},
					Destination: api.Destination{
						ID:       appName,
						Protocol: "tcp",
						Ports: api.Ports{
							Start: 8090,
							End:   8090,
						},
					},
				})
			}

			parallelRunner := &testsupport.ParallelRunner{
				NumWorkers: 4,
			}
			toDelete := make(chan (interface{}), nPolicies)

			go func() {
				parallelRunner.RunOnSlice(policies, func(policy interface{}) {
					p := policy.(api.Policy)
					do("POST", policiesUrl, p)
					toDelete <- p
				})
				close(toDelete)
			}()

			var nDeleted int32
			parallelRunner.RunOnChannel(toDelete, func(policy interface{}) {
				p := policy.(api.Policy)
				do("POST", policiesDeleteUrl, p)
				atomic.AddInt32(&nDeleted, 1)
			})

			Expect(nDeleted).To(Equal(int32(nPolicies)))

			resp := helpers.MakeAndDoRequest("GET", policiesUrl, nil, nil)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseBytes, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			var policiesResponse struct {
				TotalPolicies int          `json:"total_policies"`
				Policies      []api.Policy `json:"policies"`
			}
			Expect(json.Unmarshal(responseBytes, &policiesResponse)).To(Succeed())

			Expect(policiesResponse.TotalPolicies).To(Equal(0))
		})
	})
})
