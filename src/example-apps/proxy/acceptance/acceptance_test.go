package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Acceptance", func() {
	var (
		session    *gexec.Session
		address    string
		listenPort int
	)

	var serverIsAvailable = func() error {
		return VerifyTCPConnection(address)
	}

	Context("when no user ports are configured", func() {
		BeforeEach(func() {
			listenPort = rand.Intn(1000) + 5000
			address = fmt.Sprintf("127.0.0.1:%d", listenPort)

			exampleAppCmd := exec.Command(exampleAppPath)
			exampleAppCmd.Env = []string{
				fmt.Sprintf("PORT=%d", listenPort),
			}
			var err error
			session, err = gexec.Start(exampleAppCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(serverIsAvailable, DEFAULT_TIMEOUT).Should(Succeed())
		})

		AfterEach(func() {
			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		})
		Describe("boring server behavior", func() {
			It("should boot and gracefully terminate", func() {
				Consistently(session).ShouldNot(gexec.Exit())

				session.Interrupt()
				Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
			})
		})

		It("should respond to GET / with info", func() {
			response, err := http.DefaultClient.Get("http://" + address + "/")
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()
			Expect(response.StatusCode).To(Equal(200))

			responseBytes, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())

			var responseData struct {
				ListenAddresses []string
				Port            int
			}

			Expect(json.Unmarshal(responseBytes, &responseData)).To(Succeed())

			Expect(responseData.ListenAddresses).To(ContainElement("127.0.0.1"))
			Expect(responseData.Port).To(Equal(listenPort))
		})

		It("should respond to /proxy by proxying the request to the provided address", func() {
			response, err := http.DefaultClient.Get("http://" + address + "/proxy/example.com")
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()
			Expect(response.StatusCode).To(Equal(200))

			responseBytes, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseBytes).To(ContainSubstring("Example Domain"))
		})

		Context("when the proxy destination is invalid", func() {
			It("logs the error", func() {

				response, err := http.DefaultClient.Get("http://" + address + "/proxy/////!!")
				Expect(err).NotTo(HaveOccurred())
				defer response.Body.Close()
				Expect(response.StatusCode).To(Equal(500))

				Expect(session.Err.Contents()).To(ContainSubstring("no such host"))

			})
		})
	})

	Context("when multiple user ports are configured", func() {
		var userPorts []int

		BeforeEach(func() {
			exampleAppCmd := exec.Command(exampleAppPath)

			userPorts = []int{}
			var userPortsEnvVar string
			for i := 0; i < 5; i++ {
				userPort := rand.Intn(1000) + 5000
				userPorts = append(userPorts, userPort)
				userPortsEnvVar = fmt.Sprintf("%s%d,", userPortsEnvVar, userPort)
			}
			userPortsEnvVar = strings.TrimRight(userPortsEnvVar, ",")
			exampleAppCmd.Env = []string{
				fmt.Sprintf("PORT=%d", listenPort),
				fmt.Sprintf("USER_PORTS=%s", userPortsEnvVar),
			}
			var err error
			session, err = gexec.Start(exampleAppCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(serverIsAvailable, DEFAULT_TIMEOUT).Should(Succeed())
		})
		AfterEach(func() {
			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		})

		It("should still listen on the system provided port", func() {
			response, err := http.DefaultClient.Get("http://" + address + "/")
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()
			Expect(response.StatusCode).To(Equal(200))
		})

		It("should respond to GET / on every configured user port", func() {
			for _, port := range userPorts {
				address = fmt.Sprintf("127.0.0.1:%d", port)
				response, err := http.DefaultClient.Get("http://" + address + "/")
				Expect(err).NotTo(HaveOccurred())
				defer response.Body.Close()
				Expect(response.StatusCode).To(Equal(200))

				responseBytes, err := ioutil.ReadAll(response.Body)
				Expect(err).NotTo(HaveOccurred())

				var responseData struct {
					ListenAddresses []string
					Port            int
				}

				Expect(json.Unmarshal(responseBytes, &responseData)).To(Succeed())

				Expect(responseData.ListenAddresses).To(ContainElement("127.0.0.1"))
				Expect(responseData.Port).To(Equal(port))
			}
		})
	})

})
