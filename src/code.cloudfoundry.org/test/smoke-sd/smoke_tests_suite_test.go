package smoke_test

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	helpersConfig "github.com/cloudfoundry/cf-test-helpers/v2/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func TestSmokeTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SmokeTests Suite")
}

const Timeout_Short = 10 * time.Second
const Timeout_Cf = 4 * time.Minute

var (
	config  SmokeConfig
	appsDir string
)

type SmokeConfig struct {
	ApiEndpoint       string `json:"api"`
	AppsDomain        string `json:"apps_domain"`
	Prefix            string `json:"prefix"`
	SkipSSLValidation bool   `json:"skip_ssl_validation"`
	SmokeUser         string `json:"smoke_user"`
	SmokePassword     string `json:"smoke_password"`
	SmokeOrg          string `json:"smoke_org"`
	SmokeSpace        string `json:"smoke_space"`
}

var _ = BeforeSuite(func() {
	// Read and set config
	configPath := helpersConfig.ConfigPath()
	configBytes, err := ioutil.ReadFile(configPath)
	Expect(err).NotTo(HaveOccurred())

	err = json.Unmarshal(configBytes, &config)
	Expect(err).NotTo(HaveOccurred())

	if config.SkipSSLValidation {
		Expect(cf.Cf("api", "--skip-ssl-validation", config.ApiEndpoint).Wait(Timeout_Short)).To(gexec.Exit(0))
	} else {
		Expect(cf.Cf("api", config.ApiEndpoint).Wait(Timeout_Short)).To(gexec.Exit(0))
	}

	Auth(config.SmokeUser, config.SmokePassword)

	// Set env vars
	appsDir = os.Getenv("APPS_DIR")
	Expect(appsDir).NotTo(BeEmpty())

	rand.Seed(GinkgoRandomSeed() + int64(GinkgoParallelProcess()))
})

func Auth(username, password string) {
	By("authenticating as " + username)
	cmd := exec.Command("cf", "auth", username, password)
	sess, err := gexec.Start(cmd, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess.Wait(Timeout_Short)).Should(gexec.Exit(0))
}

func NewClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 500 * time.Millisecond,
			}).DialContext,
		},
		Timeout: 500 * time.Millisecond,
	}
}
