package smoke_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	helpersConfig "github.com/cloudfoundry/cf-test-helpers/v2/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const Timeout_Push = 5 * time.Minute

var (
	appsDir string
	config  SmokeConfig
)

type SmokeConfig struct {
	ApiEndpoint   string `json:"api"`
	AppsDomain    string `json:"apps_domain"`
	SmokeUser     string `json:"smoke_user"`
	SmokePassword string `json:"smoke_password"`
	AppInstances  int    `json:"app_instances"`
	Prefix        string `json:"prefix"`
	SmokeOrg      string `json:"smoke_org"`
}

func Auth(username, password string) {
	By("authenticating as " + username)
	cmd := exec.Command("cf", "auth", username, password)
	sess, err := gexec.Start(cmd, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess.Wait(Timeout_Short)).Should(gexec.Exit(0))
}

func TestSmoke(t *testing.T) {
	RegisterFailHandler(Fail)

	BeforeSuite(func() {
		configPath := helpersConfig.ConfigPath()
		configBytes, err := ioutil.ReadFile(configPath)
		Expect(err).NotTo(HaveOccurred())

		err = json.Unmarshal(configBytes, &config)
		Expect(err).NotTo(HaveOccurred())

		if config.AppInstances <= 0 {
			Fail("AppInstances count needs to be greater than 0")
		}

		Expect(cf.Cf("api", "--skip-ssl-validation", config.ApiEndpoint).Wait(Timeout_Short)).To(gexec.Exit(0))
		Auth(config.SmokeUser, config.SmokePassword)

		appsDir = os.Getenv("APPS_DIR")
		Expect(appsDir).NotTo(BeEmpty())

		rand.Seed(GinkgoRandomSeed() + int64(GinkgoParallelProcess()))
	})

	RunSpecs(t, "Smoke Suite")
}

func appReport(appName string, timeout time.Duration) {
	By(fmt.Sprintf("reporting app %s", appName))
	Eventually(cf.Cf("app", appName, "--guid"), timeout).Should(gexec.Exit())
	Eventually(cf.Cf("logs", appName, "--recent"), timeout).Should(gexec.Exit())
}

func appDir(appType string) string {
	return filepath.Join(appsDir, appType)
}

func defaultManifest(appType string) string {
	return filepath.Join(appDir(appType), "manifest.yml")
}

func pushApp(appName, kind string, extraArgs ...string) {
	args := append([]string{
		"push", appName,
		"-p", appDir(kind),
		"-f", defaultManifest(kind),
	}, extraArgs...)
	Expect(cf.Cf(args...).Wait(Timeout_Push)).To(gexec.Exit(0))
}

func scaleApp(appName string, instances int) {
	Expect(cf.Cf(
		"scale", appName,
		"-i", fmt.Sprintf("%d", instances),
	).Wait(Timeout_Short)).To(gexec.Exit(0))
}

func setEnv(appName, envVar, value string) {
	Expect(cf.Cf(
		"set-env", appName,
		envVar, value,
	).Wait(Timeout_Short)).To(gexec.Exit(0))
}

func start(appName string) {
	Expect(cf.Cf(
		"start", appName,
	).Wait(Timeout_Push)).To(gexec.Exit(0))
}
