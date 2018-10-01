package helpers

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"policy-server/cc_client/fixtures"
	"policy-server/config"
	"strconv"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var MockCCServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	if r.URL.Path == "/v3/spaces" {
		if r.URL.Query().Get("page") == "2" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fixtures.LiveSpacesPage2))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fixtures.LiveSpacesPage1))
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

var MockUAAServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/check_token" {
		if r.Header["Authorization"][0] == "Basic dGVzdDp0ZXN0" {
			bodyBytes, _ := ioutil.ReadAll(r.Body)
			token := strings.Split(string(bodyBytes), "=")[1]
			if len(token) == 0 {
				panic("bad token")
			}

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

func SplitUAAHostPort() (string, int) {
	url, err := url.Parse(MockUAAServer.URL)
	Expect(err).NotTo(HaveOccurred())
	UAAHost, UAAPortStr, err := net.SplitHostPort(url.Host)
	Expect(err).NotTo(HaveOccurred())
	UAAPort, err := strconv.Atoi(UAAPortStr)
	Expect(err).NotTo(HaveOccurred())
	return UAAHost, UAAPort
}

func DefaultTestConfig(dbConfig db.Config, metronAddress string, fixturesPath string) (config.Config, config.InternalConfig) {
	return DefaultTestConfigWithCCServer(dbConfig, metronAddress, fixturesPath, MockCCServer.URL)
}

func DefaultTestConfigWithCCServer(dbConfig db.Config, metronAddress string, fixturesPath string, mockCCServerURL string) (config.Config, config.InternalConfig) {
	UAAHost, UAAPort := SplitUAAHostPort()

	externalConfig := config.Config{
		ListenHost:                      "127.0.0.1",
		ListenPort:                      ports.PickAPort(),
		LogPrefix:                       "testprefix",
		DebugServerHost:                 "127.0.0.1",
		DebugServerPort:                 ports.PickAPort(),
		SkipSSLValidation:               true,
		UAAClient:                       "test",
		UAAClientSecret:                 "test",
		UAAURL:                          "http://" + UAAHost,
		UAAPort:                         UAAPort,
		CCURL:                           mockCCServerURL,
		CCCA:                            "/some/ca/cert",
		TagLength:                       1,
		Database:                        dbConfig,
		MetronAddress:                   metronAddress,
		CleanupInterval:                 60,
		CCAppRequestChunkSize:           100,
		RequestTimeout:                  10,
		MaxPolicies:                     2,
		EnableSpaceDeveloperSelfService: false,
		DatabaseMigrationTimeout:        600,
	}

	internalConfig := config.InternalConfig{
		ListenHost:                               "127.0.0.1",
		InternalListenPort:                       ports.PickAPort(),
		LogPrefix:                                "testprefix",
		DebugServerHost:                          "127.0.0.1",
		DebugServerPort:                          ports.PickAPort(),
		CACertFile:                               filepath.Join(fixturesPath, "netman-ca.crt"),
		ServerCertFile:                           filepath.Join(fixturesPath, "server.crt"),
		ServerKeyFile:                            filepath.Join(fixturesPath, "server.key"),
		TagLength:                                1,
		Database:                                 dbConfig,
		MetronAddress:                            metronAddress,
		RequestTimeout:                           10,
		EnforceExperimentalDynamicEgressPolicies: true,
	}
	return externalConfig, internalConfig
}

func WriteConfigFile(policyServerConfig interface{}) string {
	configFile, err := ioutil.TempFile("", "test-config")
	Expect(err).NotTo(HaveOccurred())

	configBytes, err := json.Marshal(policyServerConfig)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(configFile.Name(), configBytes, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	return configFile.Name()
}

func VerifyTCPConnection(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

const DEFAULT_TIMEOUT = "10s"

func StartPolicyServer(pathToBinary string, conf config.Config) *gexec.Session {
	configFilePath := WriteConfigFile(conf)

	startCmd := exec.Command(pathToBinary, "-config-file", configFilePath)
	session, err := gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
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
	return session
}

func StartInternalPolicyServer(pathToBinary string, conf config.InternalConfig) *gexec.Session {
	configFilePath := WriteConfigFile(conf)

	startCmd := exec.Command(pathToBinary, "-config-file", configFilePath)
	session, err := gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	address := fmt.Sprintf("%s:%d", conf.ListenHost, conf.InternalListenPort)
	serverIsAvailable := func() error {
		return VerifyTCPConnection(address)
	}
	debugAddress := fmt.Sprintf("%s:%d", conf.DebugServerHost, conf.DebugServerPort)
	debugServerIsAvailable := func() error {
		return VerifyTCPConnection(debugAddress)
	}
	Eventually(serverIsAvailable, DEFAULT_TIMEOUT).Should(Succeed())
	Eventually(debugServerIsAvailable, DEFAULT_TIMEOUT).Should(Succeed())
	return session
}

func MakeAndDoRequest(method string, endpoint string, extraHeaders map[string]string, body io.Reader) *http.Response {
	req, err := http.NewRequest(method, endpoint, body)
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Authorization", "Bearer valid-token")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	return resp
}

func MakeAndDoHTTPSRequest(method string, endpoint string, body io.Reader, c *tls.Config) *http.Response {
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

func RunMigrationsPreStartBinary(pathToMigrationBinary string, conf config.Config) *gexec.Session {
	configFilePath := WriteConfigFile(conf)

	startCmd := exec.Command(pathToMigrationBinary, "-config-file", configFilePath)
	session, err := gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return session
}
