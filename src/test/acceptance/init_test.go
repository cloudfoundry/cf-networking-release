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

	"code.cloudfoundry.org/go-db-helpers/testsupport"

	pusherConfig "cf-pusher/config"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

const Timeout_Push = 2 * time.Minute

var (
	appsDir    string
	config     helpers.Config
	testConfig pusherConfig.Config
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

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)

	BeforeSuite(func() {
		config = helpers.LoadConfig()

		configPath := helpers.ConfigPath()
		configBytes, err := ioutil.ReadFile(configPath)
		Expect(err).NotTo(HaveOccurred())

		err = json.Unmarshal(configBytes, &testConfig)
		Expect(err).NotTo(HaveOccurred())

		if testConfig.Applications <= 0 {
			Fail("Applications count needs to be greater than 0")
		}

		if testConfig.AppInstances <= 0 {
			Fail("AppInstances count needs to be greater than 0")
		}

		if testConfig.ProxyApplications <= 0 {
			Fail("ProxyApplications count needs to be greater than 0")
		}

		if testConfig.ProxyInstances <= 0 {
			Fail("ProxyInstances count needs to be greater than 0")
		}

		Expect(cf.Cf("api", "--skip-ssl-validation", config.ApiEndpoint).Wait(Timeout_Short)).To(gexec.Exit(0))
		AuthAsAdmin()

		appsDir = os.Getenv("APPS_DIR")
		Expect(appsDir).NotTo(BeEmpty())

		rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
	})

	RunSpecs(t, "Acceptance Suite")
}

func appDir(appType string) string {
	return filepath.Join(appsDir, appType)
}

func pushProxy(appName string) {
	Expect(cf.Cf(
		"push", appName,
		"-p", appDir("proxy"),
		"-f", defaultManifest("proxy"),
	).Wait(Timeout_Push)).To(gexec.Exit(0))
}

func defaultManifest(appType string) string {
	return filepath.Join(appDir(appType), "manifest.yml")
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

func scaleApps(apps []string, instances int) {
	parallelRunner := &testsupport.ParallelRunner{
		NumWorkers: 16,
	}
	parallelRunner.RunOnSliceStrings(apps, func(app string) {
		scaleApp(app, instances)
	})
}

func scaleApp(appName string, instances int) {
	Expect(cf.Cf(
		"scale", appName,
		"-i", fmt.Sprintf("%d", instances),
	).Wait(Timeout_Short)).To(gexec.Exit(0))
}
