package main_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"tick/a8"

	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Tick app", func() {
	var (
		registrySession *gexec.Session
		tickSession     *gexec.Session
		registryPort    int
		tickPort        int
		registryURL     string
		tickTTLSeconds  int

		startPort   int
		listenPorts int
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
			fmt.Sprintf("START_PORT=%d", startPort),
			fmt.Sprintf("LISTEN_PORTS=%d", listenPorts),
			fmt.Sprintf("PORT=%d", tickPort),
			fmt.Sprintf("REGISTRY_BASE_URL=http://127.0.0.1:%d", registryPort),
			fmt.Sprintf("REGISTRY_TTL_SECONDS=%d", tickTTLSeconds),
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
			fmt.Sprintf("A8_API_PORT=%d", registryPort),
		}
		var err error
		registrySession, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		Eventually(getURL(registryURL)).Should(MatchJSON(`{"instances": []}`))
	}

	BeforeEach(func() {
		registryPort = ports.PickAPort()
		registryURL = fmt.Sprintf("http://127.0.0.1:%d/api/v1/instances", registryPort)
		tickPort = ports.PickAPort()
		tickTTLSeconds = 11
		startPort = ports.PickAPort()
		listenPorts = 3
	})

	AfterEach(func() {
		if tickSession != nil {
			tickSession.Interrupt()
			Eventually(tickSession, DEFAULT_TIMEOUT).Should(gexec.Exit())
		}

		if registrySession != nil {
			registrySession.Interrupt()
			Eventually(registrySession, DEFAULT_TIMEOUT).Should(gexec.Exit())
		}
	})

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

	It("listens on configured ports", func() {
		StartRegistry()
		Eventually(getURL(registryURL)).Should(MatchJSON(`{"instances": []}`))

		StartTick()

		Eventually(getInstances, "5s").Should(HaveLen(1))

		for i := 0; i < listenPorts; i++ {
			_, err := getURL(fmt.Sprintf("http://%s:%d", "127.0.0.1", startPort+i))()
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("registers itself with amalgam8", func() {
		By("starting the a8 registry")
		StartRegistry()

		By("checking that registry is available and empty")
		Eventually(getURL(registryURL)).Should(MatchJSON(`{"instances": []}`))

		By("starting the tick app")
		StartTick()

		By("checking that the tick app registers itself")
		Eventually(getInstances, "5s").Should(HaveLen(1))

		By("validating the service metadata")
		instances, err := getInstances()
		Expect(err).NotTo(HaveOccurred())
		Expect(instances[0].ServiceName).To(Equal("my-tick-app"))

		By("contacting the tick app via the registered address")
		registeredAddress := instances[0].Endpoint.Value
		tickResponseBody, err := getURL(fmt.Sprintf("http://%s", registeredAddress))()
		Expect(err).NotTo(HaveOccurred())
		Expect(tickResponseBody).To(MatchJSON(`{
				"application_name": "my-tick-app",
				"instance_index": 13
			}`))

		By("verifying that the app remains registered beyond the TTL duration")
		Consistently(getInstances, "15s").Should(HaveLen(1))
	})
})

var _ = Describe("Tick error behavior", func() {
	Context("when missing a required env var", func() {
		It("fails to start", func() {
			cmd := exec.Command(binaryPath)
			cmd.Env = []string{
				"REGISTRY_BASE_URL=http://something:4001",
				`VCAP_APPLICATION={}`,
			}
			tickSession, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(tickSession).Should(gexec.Exit(1))
			Expect(tickSession.Err.Contents()).To(ContainSubstring("PORT is a required environment variable"))
		})
	})
})
