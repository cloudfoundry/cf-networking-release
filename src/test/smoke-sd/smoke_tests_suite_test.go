package smoke_test

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	helpersConfig "github.com/cloudfoundry-incubator/cf-test-helpers/config"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/gomega/gexec"
)

func TestSmokeTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SmokeTests Suite")
}

const Timeout_Short = 10 * time.Second

var (
	config  SmokeConfig
	appsDir string
)

type SmokeConfig struct {
	ApiEndpoint       string `json:"api"`
	AppsDomain        string `json:"apps_domain"`
	AdminUser         string `json:"admin_user"`
	AdminPassword     string `json:"admin_password"`
	Prefix            string `json:"prefix"`
	SkipSSLValidation bool   `json:"skip_ssl_validation"`
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

	// Log in to cf api

	if config.SkipSSLValidation {
		Expect(cf.Cf("api", "--skip-ssl-validation", config.ApiEndpoint).Wait(Timeout_Short)).To(gexec.Exit(0))
	} else {

		Expect(cf.Cf("api", config.ApiEndpoint).Wait(Timeout_Short)).To(gexec.Exit(0))
	}
	Auth(config.AdminUser, config.AdminPassword)

	// Set env vars
	appsDir = os.Getenv("APPS_DIR")
	Expect(appsDir).NotTo(BeEmpty())

	rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))

})

func AuthAsAdmin() {
	Auth(config.AdminUser, config.AdminPassword)
}

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
