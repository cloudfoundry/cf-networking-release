package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Integration", func() {
	var (
		session         *gexec.Session
		address         string
		listenPort      int
		appStartCommand *exec.Cmd
	)

	var serverIsAvailable = func() error {
		return VerifyTCPConnection(address)
	}

	BeforeEach(func() {
		listenPort = 44000 + GinkgoParallelNode()
		address = fmt.Sprintf("127.0.0.1:%d", listenPort)

		appStartCommand = exec.Command(exampleAppPath)
		appStartCommand.Env = []string{
			fmt.Sprintf("PORT=%d", listenPort),
			"DIGLETT_DESTINATION=google.com",
			"DIGLETT_FREQUENCY_MS=100",
		}
	})

	JustBeforeEach(func() {
		// Start app
		var err error
		session, err = gexec.Start(appStartCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

	})

	AfterEach(func() {
		// Stop app
		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
	})

	Describe("boring server behavior", func() {
		It("should boot and gracefully terminate", func() {
			Eventually(serverIsAvailable, DEFAULT_TIMEOUT).Should(Succeed())
			Consistently(session).ShouldNot(gexec.Exit())

			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		})

		Context("when DIGLETT_DESTINATION is not set", func() {
			BeforeEach(func() {
				appStartCommand.Env = []string{
					fmt.Sprintf("PORT=%d", listenPort),
				}
			})
			It("crashes", func() {
				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("invalid required env var DIGLETT_DESTINATION"))
			})
		})

		Context("when DIGLETT_FREQUENCY_MS is not set", func() {
			BeforeEach(func() {
				appStartCommand.Env = []string{
					fmt.Sprintf("PORT=%d", listenPort),
					"DIGLETT_DESTINATION=google.com",
				}
			})
			It("crashes", func() {
				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("invalid required env var DIGLETT_FREQUENCY_MS"))
			})
		})

		Context("when DIGLETT_FREQUENCY_MS is not a number", func() {
			BeforeEach(func() {
				appStartCommand.Env = []string{
					fmt.Sprintf("PORT=%d", listenPort),
					"DIGLETT_DESTINATION=google.com",
					"DIGLETT_FREQUENCY_MS=banana",
				}
			})
			It("crashes", func() {
				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("invalid required env var DIGLETT_FREQUENCY_MS"))
			})
		})
	})

	Describe("endpoints", func() {
		It("should respond to GET / with info", func() {
			Eventually(serverIsAvailable, DEFAULT_TIMEOUT).Should(Succeed())
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

		It("should report latency stats on /stats", func() {
			Eventually(serverIsAvailable, DEFAULT_TIMEOUT).Should(Succeed())
			// TODO
			// response, err := http.DefaultClient.Get("http://" + address + "/proxy/" + destinationAddress)
			// Expect(err).NotTo(HaveOccurred())
			// Expect(response.StatusCode).To(Equal(200))
			//
			// statsResponse, err := http.DefaultClient.Get("http://" + address + "/stats")
			// Expect(err).NotTo(HaveOccurred())
			// defer statsResponse.Body.Close()
			//
			// responseBytes, err := ioutil.ReadAll(statsResponse.Body)
			// Expect(err).NotTo(HaveOccurred())
			// var statsJSON struct {
			// 	Latency []float64
			// }
			// Expect(json.Unmarshal(responseBytes, &statsJSON)).To(Succeed())
			// Expect(len(statsJSON.Latency)).To(BeNumerically(">=", 0))
		})

	})

	Describe("Logging", func() {
		It("logs the query time and logs that the request succeeded", func() {
			Eventually(session.Out).Should(gbytes.Say("dig google.com \\d+ msec \\d+ answers"))
			Eventually(session.Out).Should(gbytes.Say("dig google.com \\d+ msec \\d+ answers"))
			Consistently(session.Out).ShouldNot(gbytes.Say("dig google.com INVALID TIME"))
			Consistently(session.Out).ShouldNot(gbytes.Say("dig google.com .* 0 answers"))
		})
		Context("when the destination address ", func() {
			BeforeEach(func() {
				appStartCommand.Env = []string{
					fmt.Sprintf("PORT=%d", listenPort),
					"DIGLETT_DESTINATION=garbageasdfasdfasdf.com",
					"DIGLETT_FREQUENCY_MS=100",
				}
			})
			It("logs the query time and logs that the request failed", func() {
				Eventually(session.Out).Should(gbytes.Say("dig garbageasdfasdfasdf.com \\d+ msec 0 answers"))
				Eventually(session.Out).Should(gbytes.Say("dig garbageasdfasdfasdf.com \\d+ msec 0 answers"))
				Consistently(session.Out).ShouldNot(gbytes.Say("dig google.com INVALID TIME"))
			})
		})

	})
})
