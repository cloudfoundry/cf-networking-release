package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"policy-server/config"
	"strings"

	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

const DEFAULT_TIMEOUT = "5s"

var policyServerPath string

var mockUAAServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/check_token" {
		if r.Header["Authorization"][0] == "Basic dGVzdDp0ZXN0" {
			bodyBytes, _ := ioutil.ReadAll(r.Body)
			token := strings.Split(string(bodyBytes), "=")[1]
			Expect(token).NotTo(BeEmpty())

			if string(token) == "valid-token" {
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

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	// only run on node 1
	fmt.Fprintf(GinkgoWriter, "building binary...")
	var err error
	policyServerPath, err = gexec.Build("policy-server/cmd/policy-server", "-race")
	fmt.Fprintf(GinkgoWriter, "done")
	Expect(err).NotTo(HaveOccurred())

	rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

func VerifyTCPConnection(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func DefaultTestConfig() config.Config {
	serverCert, err := ioutil.ReadFile("fixtures/server.crt")
	Expect(err).NotTo(HaveOccurred())
	serverKey, err := ioutil.ReadFile("fixtures/server.key")
	Expect(err).NotTo(HaveOccurred())
	caCert, err := ioutil.ReadFile("fixtures/netman-ca.crt")
	Expect(err).NotTo(HaveOccurred())

	config := config.Config{
		ListenHost:         "127.0.0.1",
		ListenPort:         9001 + GinkgoParallelNode(),
		InternalListenPort: 10001 + GinkgoParallelNode(),
		CACert:             caCert,
		ServerCert:         serverCert,
		ServerKey:          serverKey,
		UAAClient:          "test",
		UAAClientSecret:    "test",
		UAAURL:             mockUAAServer.URL,
		TagLength:          1,
	}

	return config
}

func WriteConfigFile(policyServerConfig config.Config) string {
	configFile, err := ioutil.TempFile("", "test-config")
	Expect(err).NotTo(HaveOccurred())

	configBytes, err := json.Marshal(policyServerConfig)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(configFile.Name(), configBytes, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	return configFile.Name()
}
