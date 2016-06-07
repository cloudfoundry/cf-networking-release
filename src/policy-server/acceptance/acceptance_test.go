package acceptance_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"policy-server/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Acceptance", func() {
	var (
		session       *gexec.Session
		conf          config.Config
		address       string
		mockUAAServer *httptest.Server
	)

	var serverIsAvailable = func() error {
		return VerifyTCPConnection(address)
	}

	BeforeEach(func() {
		mockUAAServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/check_token" {
				if r.Header["Authorization"][0] == "Basic dGVzdDp0ZXN0Cg==" {
					token, err := ioutil.ReadAll(r.Body)
					Expect(err).NotTo(HaveOccurred())

					if string(token) == "token=valid-token" {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"scope":["network.admin"], "user_name":"some-user"}`))
					} else {
						w.WriteHeader(http.StatusBadRequest)
						w.Write([]byte(`{"error_description":"Some requested scopes are missing: network.admin"}`))
					}
				} else {
					w.WriteHeader(http.StatusUnauthorized)
				}
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))

		conf = config.Config{
			ListenHost:      "127.0.0.1",
			ListenPort:      9001 + GinkgoParallelNode(),
			UAAClient:       "test",
			UAAClientSecret: "test",
			UAAURL:          mockUAAServer.URL,
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
	})

	It("should boot and gracefully terminate", func() {
		Consistently(session).ShouldNot(gexec.Exit())

		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
	})

	Describe("adding policies", func() {
		It("has an available endpoint", func() {
			client := &http.Client{}

			resp, err := client.Post(fmt.Sprintf("http://%s:%d/rule", conf.ListenHost, conf.ListenPort), "", bytes.NewReader([]byte{}))
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		PIt("has a whoami endpoint", func() {
			client := &http.Client{}
			tokenString := "token=valid-token"
			req, err := http.NewRequest("GET", fmt.Sprintf("http://%s:%d/networking/v0/external/whoami", conf.ListenHost, conf.ListenPort), bytes.NewBuffer([]byte{}))
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenString))

			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(responseString).To(ContainSubstring("some-user"))
		})
	})
})
