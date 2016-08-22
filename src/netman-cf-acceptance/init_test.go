package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var (
	appsDir    string
	config     helpers.Config
	testConfig struct {
		TestUser         string `json:"test_user"`
		TestUserPassword string `json:"test_user_password"`
		Applications     int    `json:"reflex_applications"`
		AppInstances     int    `json:"reflex_instances"`
	}
	preBuiltBinaries map[string]string
)

func Auth(username, password string) {
	By("authenticating as " + username)
	cmd := exec.Command("cf", "auth", username, password)
	sess, err := gexec.Start(cmd, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess.Wait(Timeout_Short)).Should(gexec.Exit(0))
}

func AuthAsAdmin() {
	Auth(config.AdminUser, config.AdminPassword)
}

func preBuildLinuxBinary(appType string) {
	By("pre-building the linux binary for " + appType)
	os.Setenv("GOOS", "linux")
	os.Setenv("GOARCH", "amd64")
	appDir := filepath.Join(appsDir, appType)
	Expect(exec.Command("go", "build", "-o", filepath.Join(appDir, appType), appDir).Run()).To(Succeed())
}

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)

	BeforeSuite(func() {
		config = helpers.LoadConfig()

		configPath := helpers.ConfigPath()
		configBytes, err := ioutil.ReadFile(configPath)
		Expect(err).NotTo(HaveOccurred())

		err = json.Unmarshal(configBytes, &testConfig)
		Expect(err).NotTo(HaveOccurred())

		Expect(cf.Cf("api", "--skip-ssl-validation", config.ApiEndpoint).Wait(Timeout_Short)).To(gexec.Exit(0))
		AuthAsAdmin()

		appsDir = os.Getenv("APPS_DIR")
		Expect(appsDir).NotTo(BeEmpty())

		preBuildLinuxBinary("proxy")
		preBuildLinuxBinary("reflex")

		rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
	})

	AfterSuite(func() {
		// remove binaries
		Expect(os.Remove(filepath.Join(appsDir, "proxy", "proxy"))).To(Succeed())
		Expect(os.Remove(filepath.Join(appsDir, "reflex", "reflex"))).To(Succeed())
	})

	RunSpecs(t, "Acceptance Suite")
}

func appDir(appType string) string {
	return filepath.Join(appsDir, appType)
}

func pushApp(appName string) {
	pushAppOfType(appName, "proxy")
}

func pushAppsOfType(appNames []string, appType string) {
	for _, app := range appNames {
		By(fmt.Sprintf("pushing app %s of type %s", app, appType))
		pushAppOfType(app, appType)
	}
}

func pushAppOfType(appName, appType string) {
	Expect(cf.Cf(
		"push", appName,
		"-p", appDir(appType),
		"-f", filepath.Join(appDir(appType), "manifest.yml"),
		"-c", "./"+appType,
		"-b", "binary_buildpack",
	).Wait(Timeout_Push)).To(gexec.Exit(0))
}

func scaleApps(apps []string, instances int) {
	for _, app := range apps {
		scaleApp(app, instances)
	}
}

func scaleApp(appName string, instances int) {
	Expect(cf.Cf(
		"scale", appName,
		"-i", fmt.Sprintf("%d", instances),
	).Wait(Timeout_Short)).To(gexec.Exit(0))
}

func appsReport(appNames []string, timeout time.Duration) {
	for _, app := range appNames {
		appReport(app, timeout)
	}
}

func appReport(appName string, timeout time.Duration) {
	By(fmt.Sprintf("reporting app %s", appName))
	Eventually(cf.Cf("app", appName, "--guid"), timeout).Should(gexec.Exit())
	Eventually(cf.Cf("logs", appName, "--recent"), timeout).Should(gexec.Exit())
}
