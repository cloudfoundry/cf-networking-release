package integration_test

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"policy-server/cc_client/fixtures"
	"policy-server/config"
	"strconv"
	"strings"

	"code.cloudfoundry.org/go-db-helpers/db"
	"code.cloudfoundry.org/go-db-helpers/metrics"

	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/types"

	"testing"
)

const DEFAULT_TIMEOUT = "5s"

var policyServerPath string

var HaveName = func(name string) types.GomegaMatcher {
	return WithTransform(func(ev metrics.Event) string {
		return ev.Name
	}, Equal(name))
}

var mockCCServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.Header["Authorization"][0] != "bearer valid-token" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if r.URL.Path == "/v3/apps" {
		if strings.Contains(r.URL.RawQuery, "app-guid-not-in-my-spaces") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fixtures.AppsV3TwoSpaces))
			return
		}
		if strings.Contains(r.URL.RawQuery, "live-app-1-guid") && !strings.Contains(r.URL.RawQuery, "live-app-2-guid") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fixtures.AppsV3LiveApp1GUID))
			return
		}
		if strings.Contains(r.URL.RawQuery, "live-app-2-guid") && !strings.Contains(r.URL.RawQuery, "live-app-1-guid") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fixtures.AppsV3LiveApp2GUID))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fixtures.AppsV3OneSpace))
		return
	}

	if r.URL.Path == "/v2/spaces/space-1-guid" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fixtures.Space1))
		return
	}
	if r.URL.Path == "/v2/spaces/space-2-guid" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fixtures.Space2))
		return
	}

	if r.URL.Path == "/v2/spaces" {
		if strings.Contains(r.URL.RawQuery, "space-1") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fixtures.UserSpace))
			return
		}
		if strings.Contains(r.URL.RawQuery, "space-2") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fixtures.UserSpaceEmpty))
			return
		}
	}

	if r.URL.Path == "/v2/users/some-user-id/spaces" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fixtures.UserSpaces))
		return
	}

	w.WriteHeader(http.StatusTeapot)
	return
}))

var mockUAAServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/check_token" {
		if r.Header["Authorization"][0] == "Basic dGVzdDp0ZXN0" {
			bodyBytes, _ := ioutil.ReadAll(r.Body)
			token := strings.Split(string(bodyBytes), "=")[1]
			Expect(token).NotTo(BeEmpty())

			switch token {
			case "valid-token":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"scope":["network.admin"], "user_name":"some-user", "user_id": "some-user-id"}`))
			case "space-dev-with-network-write-token":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"scope":["network.write"], "user_name":"some-user", "user_id": "some-user-id"}`))
			case "space-dev-token":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"scope":[], "user_name":"some-user", "user_id": "some-user-id"}`))
			default:
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error_description":"banana"}`))
			}
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
		return
	}

	if r.URL.Path == "/oauth/token" {
		token := `
		{
  "access_token" : "valid-token",
  "token_type" : "bearer",
  "refresh_token" : "valid-token-r",
  "expires_in" : 43199,
  "scope" : "scim.userids openid cloud_controller.read password.write cloud_controller.write",
  "jti" : "9796365e7c364f41a9d2436aef6b8351"
}
		`
		if r.Header["Authorization"][0] == "Basic dGVzdDp0ZXN0" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(token))
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

var _ = SynchronizedBeforeSuite(func() []byte {
	fmt.Fprintf(GinkgoWriter, "building binary...")
	policyServerPath, err := gexec.Build("policy-server/cmd/policy-server", "-race")
	fmt.Fprintf(GinkgoWriter, "done")
	Expect(err).NotTo(HaveOccurred())

	return []byte(policyServerPath)
}, func(data []byte) {
	policyServerPath = string(data)
	rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = SynchronizedAfterSuite(func() {}, func() {
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

func DefaultTestConfig(dbConfig db.Config, metronAddress string) config.Config {
	UAAHost, UAAPort := SplitUAAHostPort()

	config := config.Config{
		ListenHost:            "127.0.0.1",
		ListenPort:            10001 + GinkgoParallelNode(),
		InternalListenPort:    20001 + GinkgoParallelNode(),
		DebugServerHost:       "127.0.0.1",
		DebugServerPort:       30001 + GinkgoParallelNode(),
		CACertFile:            "fixtures/netman-ca.crt",
		ServerCertFile:        "fixtures/server.crt",
		ServerKeyFile:         "fixtures/server.key",
		SkipSSLValidation:     true,
		UAAClient:             "test",
		UAAClientSecret:       "test",
		UAAURL:                "http://" + UAAHost,
		UAAPort:               UAAPort,
		CCURL:                 mockCCServer.URL,
		TagLength:             1,
		Database:              dbConfig,
		MetronAddress:         metronAddress,
		CleanupInterval:       60,
		CCAppRequestChunkSize: 100,
		RequestTimeout:        10,
	}
	return config
}

func SplitUAAHostPort() (string, int) {
	url, err := url.Parse(mockUAAServer.URL)
	Expect(err).NotTo(HaveOccurred())
	UAAHost, UAAPortStr, err := net.SplitHostPort(url.Host)
	Expect(err).NotTo(HaveOccurred())
	UAAPort, err := strconv.Atoi(UAAPortStr)
	Expect(err).NotTo(HaveOccurred())
	return UAAHost, UAAPort
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

func makeAndDoRequest(method string, endpoint string, body io.Reader) *http.Response {
	req, err := http.NewRequest(method, endpoint, body)
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Authorization", "Bearer valid-token")
	resp, err := http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	return resp
}

func makeAndDoHTTPSRequest(method string, endpoint string, body io.Reader, c *tls.Config) *http.Response {
	req, err := http.NewRequest(method, endpoint, body)
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Authorization", "Bearer valid-token")
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: c,
		},
	}
	resp, err := client.Do(req)
	Expect(err).NotTo(HaveOccurred())
	return resp
}

func configurePolicyServers(template config.Config, instances int) []config.Config {
	var configs []config.Config
	for i := 0; i < instances; i++ {
		conf := template
		conf.ListenPort += i * 100
		conf.InternalListenPort += i * 100
		conf.DebugServerPort += i * 100
		configs = append(configs, conf)
	}
	return configs
}

func startPolicyServers(configs []config.Config) []*gexec.Session {
	var sessions []*gexec.Session
	for _, conf := range configs {
		configFilePath := WriteConfigFile(conf)

		policyServerCmd := exec.Command(policyServerPath, "-config-file", configFilePath)
		session, err := gexec.Start(policyServerCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		address := fmt.Sprintf("%s:%d", conf.ListenHost, conf.ListenPort)
		serverIsAvailable := func() error {
			return VerifyTCPConnection(address)
		}
		debugAddress := fmt.Sprintf("%s:%d", conf.DebugServerHost, conf.DebugServerPort)
		debugServerIsAvailable := func() error {
			return VerifyTCPConnection(debugAddress)
		}
		Eventually(serverIsAvailable, DEFAULT_TIMEOUT).Should(Succeed())
		Eventually(debugServerIsAvailable, DEFAULT_TIMEOUT).Should(Succeed())

		sessions = append(sessions, session)
	}
	return sessions
}

func stopPolicyServers(sessions []*gexec.Session) {
	for _, session := range sessions {
		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
	}
}

func policyServerUrl(route string, confs []config.Config) string {
	conf := confs[rand.Intn(len(confs))]
	return fmt.Sprintf("http://%s:%d/networking/v0/%s", conf.ListenHost, conf.ListenPort, route)
}
