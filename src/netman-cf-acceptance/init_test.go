package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"lib/testsupport"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"

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
		Applications     int    `json:"test_applications"`
		AppInstances     int    `json:"test_app_instances"`
		Policies         int    `json:"test_policies"`
		ProxyInstances   int    `json:"proxy_instances"`
	}
	preBuiltBinaries map[string]string
)

type TickManifest struct {
	Applications []struct {
		Name      string `yaml:"name"`
		Memory    string `yaml:"memory"`
		DiskQuota string `yaml:"disk_quota"`
		BuildPack string `yaml:"buildpack"`
		Instances int    `yaml:"instances"`
		Env       struct {
			GoPackageName   string `yaml:"GOPACKAGENAME"`
			RegistryBaseURL string `yaml:"REGISTRY_BASE_URL"`
			StartPort       int    `yaml:"START_PORT"`
			ListenPorts     int    `yaml:"LISTEN_PORTS"`
		} `yaml:"env"`
	}
}

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

func preBuildRegistry(appType, path string) {
	By("pre-building the linux binary for " + appType)
	os.Setenv("GOOS", "linux")
	os.Setenv("GOARCH", "amd64")
	appDir := filepath.Join(appsDir, appType)
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		os.Mkdir(appDir, os.ModePerm)
	}
	Expect(exec.Command("go", "build", "-o", filepath.Join(appDir, appType), path).Run()).To(Succeed())
}

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)

	BeforeSuite(func() {
		config = helpers.LoadConfig()

		configPath := helpers.ConfigPath()
		configBytes, err := ioutil.ReadFile(configPath)
		Expect(err).NotTo(HaveOccurred())

		//default to 1 policy if not otherwise configured
		testConfig.Policies = 1
		testConfig.ProxyInstances = 1
		err = json.Unmarshal(configBytes, &testConfig)
		Expect(err).NotTo(HaveOccurred())

		if testConfig.Applications <= 0 {
			Fail("Applications count needs to be greater than 0")
		}

		if testConfig.AppInstances <= 0 {
			Fail("AppInstances count needs to be greater than 0")
		}

		Expect(cf.Cf("api", "--skip-ssl-validation", config.ApiEndpoint).Wait(Timeout_Short)).To(gexec.Exit(0))
		AuthAsAdmin()

		appsDir = os.Getenv("APPS_DIR")
		Expect(appsDir).NotTo(BeEmpty())

		preBuildLinuxBinary("proxy")
		preBuildLinuxBinary("tick")
		preBuildRegistry("registry", "../github.com/amalgam8/amalgam8/cmd/registry/")

		rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
	})

	AfterSuite(func() {
		// remove binaries
		Expect(os.Remove(filepath.Join(appsDir, "proxy", "proxy"))).To(Succeed())
		Expect(os.Remove(filepath.Join(appsDir, "tick", "tick"))).To(Succeed())
		Expect(os.RemoveAll(filepath.Join(appsDir, "registry"))).To(Succeed())
	})

	RunSpecs(t, "Acceptance Suite")
}

func modifyTickManifest(registryName string) string {
	manifestFile := defaultManifest("tick")

	var manifestStruct TickManifest
	manifestBytes, err := ioutil.ReadFile(manifestFile)

	Expect(yaml.Unmarshal(manifestBytes, &manifestStruct)).To(Succeed())
	manifestStruct.Applications[0].Instances = 1
	manifestStruct.Applications[0].Env.RegistryBaseURL = "http://" + registryName + "." + config.AppsDomain
	manifestStruct.Applications[0].Env.StartPort = 7000
	manifestStruct.Applications[0].Env.ListenPorts = testConfig.Policies

	manifestBytes, err = yaml.Marshal(manifestStruct)
	Expect(err).NotTo(HaveOccurred())

	tempDir, err := ioutil.TempDir("", "")
	Expect(err).NotTo(HaveOccurred())

	newManifestFile := filepath.Join(tempDir, "test.yml")
	Expect(ioutil.WriteFile(newManifestFile, manifestBytes, os.ModePerm)).To(Succeed())
	return newManifestFile
}

func appDir(appType string) string {
	return filepath.Join(appsDir, appType)
}

func pushApp(appName string) {
	pushAppsOfType([]string{appName}, "proxy", defaultManifest("proxy"))
}

func defaultManifest(appType string) string {
	return filepath.Join(appDir(appType), "manifest.yml")
}

func pushAppsOfType(appNames []string, appType string, manifest string) {
	By(fmt.Sprintf("pushing %d apps of type %s", len(appNames), appType))

	parallelRunner := &testsupport.ParallelRunner{
		NumWorkers: 16,
	}
	parallelRunner.RunOnSliceStrings(appNames, func(appName string) {
		pushAppOfType(appName, appType, manifest)
	})
}

func pushAppOfType(appName string, appType string, manifest string) {
	By(fmt.Sprintf("pushing app %s of type %s", appName, appType))
	Expect(cf.Cf(
		"push", appName,
		"-p", appDir(appType),
		"-f", manifest,
		"-c", "./"+appType,
		"-b", "binary_buildpack",
	).Wait(Timeout_Push)).To(gexec.Exit(0))
}

func pushRegistryApp(appName string) {
	Expect(cf.Cf(
		"push", appName,
		"-p", appDir("registry"),
		"-c", "./registry",
		"-b", "binary_buildpack",
		"-m", "32M",
	).Wait(Timeout_Push)).To(gexec.Exit(0))
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
