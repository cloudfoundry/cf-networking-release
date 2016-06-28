package acceptance_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"lib/testsupport"
	"math/rand"
	"net/http"
	"os/exec"
	"policy-server/config"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Acceptance", func() {
	var (
		session      *gexec.Session
		conf         config.Config
		address      string
		testDatabase *testsupport.TestDatabase
	)

	var serverIsAvailable = func() error {
		return VerifyTCPConnection(address)
	}

	BeforeEach(func() {
		dbName := fmt.Sprintf("test_netman_database_%x", rand.Int())
		dbConnectionInfo := testsupport.GetDBConnectionInfo()
		testDatabase = dbConnectionInfo.CreateDatabase(dbName)

		conf = config.Config{
			ListenHost:      "127.0.0.1",
			ListenPort:      9001 + GinkgoParallelNode(),
			UAAClient:       "test",
			UAAClientSecret: "test",
			UAAURL:          mockUAAServer.URL,
			Database:        testDatabase.DBConfig(),
			TagLength:       1,
		}
		configFilePath := WriteConfigFile(conf)

		policyServerCmd := exec.Command(policyServerPath, "-config-file", configFilePath)
		var err error
		session, err = gexec.Start(policyServerCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		address = fmt.Sprintf("%s:%d", conf.ListenHost, conf.ListenPort)

		Eventually(serverIsAvailable, DEFAULT_TIMEOUT).Should(Succeed())
	})

	AfterEach(func() {
		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())

		if testDatabase != nil {
			testDatabase.Destroy()
		}
	})

	Describe("boring server behavior", func() {
		It("should boot and gracefully terminate", func() {
			Consistently(session).ShouldNot(gexec.Exit())

			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		})

		It("responds with uptime when accessed on the root path", func() {
			req, err := http.NewRequest("GET", fmt.Sprintf("http://%s:%d/", conf.ListenHost, conf.ListenPort), nil)
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(responseString).To(ContainSubstring("Network policy server, up for"))
		})

		It("responds with uptime when accessed on the context path", func() {
			req, err := http.NewRequest("GET", fmt.Sprintf("http://%s:%d/networking", conf.ListenHost, conf.ListenPort), nil)
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(responseString).To(ContainSubstring("Network policy server, up for"))
		})

		It("has a whoami endpoint", func() {
			resp := makeAndDoRequest(
				"GET",
				fmt.Sprintf("http://%s:%d/networking/v0/external/whoami", conf.ListenHost, conf.ListenPort),
				nil,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(ContainSubstring("some-user"))
		})
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
			Entry("DELETE to policies",
				"DELETE",
				"networking/v0/external/policies",
				`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`,
			),
		)
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
			Expect(responseString).To(MatchJSON(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`))
		})
	})

	Describe("listing policies", func() {
		Context("when providing a list of ids as a query parameter", func() {
			It("responds with a 200 and lists all policies which contain one of those ids", func() {
				body := strings.NewReader(`{ "policies": [
				 {"source": { "id": "app1" }, "destination": { "id": "app2", "protocol": "tcp", "port": 8080 } },
				 {"source": { "id": "app3" }, "destination": { "id": "app1", "protocol": "tcp", "port": 9999 } },
				 {"source": { "id": "app3" }, "destination": { "id": "app4", "protocol": "tcp", "port": 3333 } }
				 ]}
				`)
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
					fmt.Sprintf("http://%s:%d/networking/v0/external/policies?id=app1,app2", conf.ListenHost, conf.ListenPort),
					nil,
				)

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err = ioutil.ReadAll(resp.Body)
				Expect(responseString).To(MatchJSON(`{ "policies": [
				 {"source": { "id": "app1" }, "destination": { "id": "app2", "protocol": "tcp", "port": 8080 } },
				 {"source": { "id": "app3" }, "destination": { "id": "app1", "protocol": "tcp", "port": 9999 } }
				 ]}
				`))
			})
		})
	})

	Describe("deleting policies", func() {
		BeforeEach(func() {
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
		})

		Context("when all of the deletes succeed", func() {
			It("responds with 200 and a body of {} and we can see it is removed from the list", func() {
				body := strings.NewReader(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`)
				resp := makeAndDoRequest(
					"DELETE",
					fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
					body,
				)

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(MatchJSON(`{}`))

				resp = makeAndDoRequest(
					"GET",
					fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
					nil,
				)

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err = ioutil.ReadAll(resp.Body)
				Expect(responseString).To(MatchJSON(`{ "policies": [] }`))
			})
		})

		Context("when one of the policies to delete does not exist", func() {
			It("responds with status 200", func() {
				body := strings.NewReader(`{ "policies": [
						{"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } },
						{"source": { "id": "some-non-existent-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } }
					] }`)
				resp := makeAndDoRequest(
					"DELETE",
					fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
					body,
				)

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(MatchJSON(`{}`))
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
				"DELETE",
				fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
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
		})
	})
})

func makeAndDoRequest(method string, endpoint string, body io.Reader) *http.Response {
	req, err := http.NewRequest(method, endpoint, body)
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Authorization", "Bearer valid-token")
	resp, err := http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	return resp
}
