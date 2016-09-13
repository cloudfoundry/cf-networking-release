package main_test

import (
	"encoding/json"
	"example-apps/tick/a8"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os/exec"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Tick", func() {
	var (
		registrySession *gexec.Session
		tickSession     *gexec.Session
		registryPort    string
		tickPort        string
		tickURL         string
		registryURL     string
	)

	var getURL = func(url string) func() (string, error) {
		return func() (string, error) {
			resp, err := http.Get(url)
			if err != nil {
				return "", err
			}
			respBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return "", err
			}
			if resp.StatusCode != http.StatusOK {
				return "", fmt.Errorf("unexpected status code %d: %s",
					resp.StatusCode, string(respBytes))
			}

			return string(respBytes), nil
		}
	}

	var StartTick = func() {
		cmd := exec.Command(binaryPath)
		cmd.Env = []string{
			fmt.Sprintf("PORT=%s", tickPort),
			fmt.Sprintf("REGISTRY_BASE_URL=http://127.0.0.1:%s", registryPort),
			fmt.Sprintf(`VCAP_APPLICATION={
				"instance_index": 13,
				"application_name": "my-tick-app"
			}`),
		}
		var err error
		tickSession, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	}

	var StartRegistry = func() {
		cmd := exec.Command(registryBinaryPath)
		cmd.Env = []string{
			fmt.Sprintf("A8_API_PORT=%s", registryPort),
		}
		var err error
		registrySession, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		registryPort = strconv.Itoa(40000 + rand.Intn(20000))
		registryURL = fmt.Sprintf("http://127.0.0.1:%s/api/v1/instances", registryPort)
		tickPort = strconv.Itoa(40000 + rand.Intn(20000))
		tickURL = fmt.Sprintf("http://127.0.0.1:%s", tickPort)

		StartRegistry()

		By("checking that registry is available and empty")
		Eventually(getURL(registryURL)).Should(MatchJSON(`{"instances": []}`))
	})

	AfterEach(func() {
		if tickSession != nil {
			tickSession.Interrupt()
			Eventually(tickSession, DEFAULT_TIMEOUT).Should(gexec.Exit())
		}
	})

	Describe("boring daemon behavior", func() {
		It("should boot and gracefully terminate", func() {
			StartTick()
			Consistently(tickSession).ShouldNot(gexec.Exit())
		})
	})

	Describe("HTTP server", func() {
		It("listens on PORT env var", func() {
			StartTick()

			Eventually(getURL(tickURL)).Should(MatchJSON(`{
				"application_name": "my-tick-app",
				"instance_index": 13
			}`))
		})

		Context("when PORT env variable is missing", func() {
			It("fails to start", func() {
				tickPort = ""
				StartTick()
				Eventually(tickSession).Should(gexec.Exit(1))
				Expect(tickSession.Err.Contents()).To(ContainSubstring("PORT is a required environment variable"))
			})
		})
	})

	Describe("Registry", func() {
		var getInstances = func() ([]a8.ServiceInstance, error) {
			responseBody, err := getURL(registryURL)()
			if err != nil {
				return nil, err
			}
			var instancesResponse struct {
				Instances []a8.ServiceInstance `json:"instances"`
			}
			Expect(json.Unmarshal([]byte(responseBody), &instancesResponse)).To(Succeed())
			return instancesResponse.Instances, nil
		}

		It("supports registration of the tick app", func() {
			StartTick()

			By("checking that the tick app registers itself")
			Eventually(getInstances).Should(HaveLen(1))

			By("validating the service metadata")
			instances, err := getInstances()
			Expect(err).NotTo(HaveOccurred())
			Expect(instances[0].ServiceName).To(Equal("my-tick-app"))
		})
	})

})
