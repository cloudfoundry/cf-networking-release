package main_test

import (
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
	)

	var StartTick = func() {
		cmd := exec.Command(binaryPath)
		cmd.Env = []string{
			fmt.Sprintf("PORT=%s", tickPort),
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
		tickPort = strconv.Itoa(40000 + rand.Intn(20000))
		tickURL = fmt.Sprintf("http://127.0.0.1:%s", tickPort)

		StartRegistry()
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

	var _ = Describe("HTTP server", func() {
		Context("when PORT env variable is missing", func() {

			It("server does not start", func() {
				tickPort = ""
				StartTick()
				Eventually(tickSession).Should(gexec.Exit(1))
				Expect(tickSession.Err.Contents()).To(ContainSubstring("missing required env var PORT"))
			})
		})

		It("listens on PORT env var", func() {
			StartTick()

			Eventually(func() (string, error) {
				resp, err := http.Get(tickURL)
				if err != nil {
					return "", err
				}
				respBytes, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return "", err
				}

				return string(respBytes), nil
			}).Should(Equal("hello"))

		})
	})

	var _ = Describe("Registry", func() {

		It("tick registers itself on startup", func() {

		})
	})

})
