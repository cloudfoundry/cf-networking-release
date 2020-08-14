package acceptance_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/testsupport"

	"cf-pusher/cf_cli_adapter"
	pusherConfig "cf-pusher/config"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	helpers "github.com/cloudfoundry-incubator/cf-test-helpers/config"
	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const Timeout_Push = 2 * time.Minute

var (
	appsDir    string
	config     *helpers.Config
	testConfig pusherConfig.Config
	cfCLI      *cf_cli_adapter.Adapter
)

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)

	BeforeSuite(func() {
		cfCLI = cf_cli_adapter.NewAdapter()
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

func Auth(username, password string) {
	By("authenticating as " + username)
	cmd := exec.Command("cf", "auth", username, password)
	sess, err := gexec.Start(cmd, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess.Wait(Timeout_Short)).Should(gexec.Exit(0))
}

func getUAABaseURL() string {
	sess := cf.Cf("curl", "/v2/info")
	Eventually(sess.Wait(Timeout_Short)).Should(gexec.Exit(0))
	var response struct {
		TokenEndpoint string `json:"token_endpoint"`
	}
	err := json.Unmarshal(sess.Out.Contents(), &response)
	Expect(err).NotTo(HaveOccurred())

	uaaBaseURL := response.TokenEndpoint
	Expect(uaaBaseURL).To(HavePrefix("https://uaa."))
	return uaaBaseURL
}

func AuthAsAdmin() {
	Auth(config.AdminUser, config.AdminPassword)
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

func pushAppWithInstanceCount(appName string, appCount int) {
	Expect(cf.Cf(
		"push", appName,
		"-p", appDir("proxy"),
		"-i", fmt.Sprintf("%d", appCount),
		"-f", defaultManifest("proxy"),
	).Wait(Timeout_Push)).To(gexec.Exit(0))

	waitForAllInstancesToBeRunning(appName)
}

func waitForAllInstancesToBeRunning(appName string) {
	appGuidSession := cf.Cf("app", appName, "--guid")
	Expect(appGuidSession.Wait(Timeout_Short)).To(gexec.Exit(0))

	capiURL := fmt.Sprintf("v2/apps/%s/instances", strings.TrimSpace(string(appGuidSession.Out.Contents())))

	type instanceInfo struct {
		State string `json:"state"`
	}

	instances := make(map[string]instanceInfo)

	allInstancesRunning := func() bool {
		session := cf.Cf("curl", capiURL)
		Expect(session.Wait(Timeout_Short)).To(gexec.Exit(0))

		json.Unmarshal(session.Out.Contents(), &instances)
		Expect(instances).To(Not(BeEmpty()))

		for _, instance := range instances {
			if instance.State != "RUNNING" {
				return false
			}
		}
		return true
	}

	Eventually(allInstancesRunning, "30s", "500ms").Should(Equal(true), "not all instances running")
}

func restage(appName string) {
	Expect(cf.Cf(
		"restage", appName,
	).Wait(Timeout_Push)).To(gexec.Exit(0))

	waitForAllInstancesToBeRunning(appName)
}

type AppInstance struct {
	hostIdentifier string
	index          string
	internalIP     string
}

func getAppInstances(appName string, instances int) []AppInstance {
	apps := make([]AppInstance, instances)
	for i := 0; i < instances; i++ {
		session := cf.Cf("ssh", appName, "-i", fmt.Sprintf("%d", i), "-c", "env | grep CF_INSTANCE")
		Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))

		env := strings.Split(string(session.Out.Contents()), "\n")
		var app AppInstance
		for _, envVar := range env {
			kv := strings.Split(envVar, "=")
			switch kv[0] {
			case "CF_INSTANCE_IP":
				app.hostIdentifier = kv[1]
			case "CF_INSTANCE_INDEX":
				app.index = kv[1]
			case "CF_INSTANCE_INTERNAL_IP":
				app.internalIP = kv[1]
			}
		}
		apps[i] = app
	}
	return apps
}

func findTwoInstancesOnTheSameHost(apps []AppInstance) (AppInstance, AppInstance) {
	hostsToApps := map[string]AppInstance{}

	for _, app := range apps {
		foundApp, ok := hostsToApps[app.hostIdentifier]
		if ok {
			return foundApp, app
		}
		hostsToApps[app.hostIdentifier] = app
	}

	Expect(errors.New("failed to find two instances on the same host")).ToNot(HaveOccurred())
	return AppInstance{}, AppInstance{}
}

func findTwoInstancesOnDifferentHosts(apps []AppInstance) (AppInstance, AppInstance) {
	for _, app := range apps[1:] {
		if apps[0].hostIdentifier != app.hostIdentifier {
			return apps[0], app
		}
	}

	Expect(errors.New("failed to find two instances on different hosts")).ToNot(HaveOccurred())
	return AppInstance{}, AppInstance{}
}
