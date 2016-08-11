package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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
	binaryPath, err := gexec.Build(appDir)
	Expect(err).NotTo(HaveOccurred())
	Expect(os.Rename(binaryPath, filepath.Join(appDir, appType))).To(Succeed())
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

func pushAppOfType(appName, appType string) {
	Expect(cf.Cf(
		"push", appName,
		"-p", appDir(appType),
		"-f", filepath.Join(appDir(appType), "manifest.yml"),
		"-c", "./"+appType,
		"-b", "binary_buildpack",
	).Wait(Timeout_Push)).To(gexec.Exit(0))
}

func scaleApp(appName string, instances int) {
	Expect(cf.Cf(
		"scale", appName,
		"-i", fmt.Sprintf("%d", instances),
	).Wait(Timeout_Short)).To(gexec.Exit(0))

	// wait for ssh to become available on new instances
	time.Sleep(15 * time.Second)
}

const (
	ip4Regex         = `(?:[0-9]{1,3}\.){3}[0-9]{1,3}`
	ipAddrParseRegex = `inet (` + ip4Regex + `)/24 scope global eth0`
)

func getInstanceIP(appName string, instanceIndex int) string {
	sshSession := cf.Cf(
		"ssh", appName,
		"-i", fmt.Sprintf("%d", instanceIndex),
		"--skip-host-validation",
		"-c", "ip addr",
	)
	Expect(sshSession.Wait(2 * Timeout_Short)).To(gexec.Exit(0))

	addrOut := string(sshSession.Out.Contents())
	matches := regexp.MustCompile(ipAddrParseRegex).FindStringSubmatch(addrOut)
	return matches[1]
}

func curlFromApp(appName string, instanceIndex int, endpoint string, expectSuccess bool) string {
	var output string

	tryIt := func() int {
		sshSession := cf.Cf(
			"ssh", appName,
			"-i", fmt.Sprintf("%d", instanceIndex),
			"--skip-host-validation",
			"-c", fmt.Sprintf("curl --connect-timeout 5 %s", endpoint),
		)
		Expect(sshSession.Wait(2 * Timeout_Short)).To(gexec.Exit())
		output = string(sshSession.Out.Contents())
		return sshSession.ExitCode()
	}

	if expectSuccess {
		Eventually(tryIt).Should(Equal(0))
	} else {
		Eventually(func() bool {
			code := tryIt()
			const CURL_EXIT_CODE_COULDNT_RESOLVE_HOST = 6
			const CURL_EXIT_CODE_COULDNT_CONNECT = 7
			const CURL_EXIT_CODE_OPERATION_TIMEDOUT = 28
			switch code {
			case CURL_EXIT_CODE_COULDNT_RESOLVE_HOST, CURL_EXIT_CODE_COULDNT_CONNECT, CURL_EXIT_CODE_OPERATION_TIMEDOUT:
				return true
			default:
				fmt.Printf("curl exit code: %d\n", code)
				return false
			}
		}).Should(BeTrue())
	}
	return output
}

func getAppGuid(appName string) string {
	session := cf.Cf("app", appName, "--guid")
	Expect(session.Wait(Timeout_Short)).To(gexec.Exit(0))
	return strings.TrimSpace(string(session.Out.Contents()))
}

func AppReport(appName string, timeout time.Duration) {
	Eventually(cf.Cf("app", appName, "--guid"), timeout).Should(gexec.Exit())
	Eventually(cf.Cf("logs", appName, "--recent"), timeout).Should(gexec.Exit())
}
