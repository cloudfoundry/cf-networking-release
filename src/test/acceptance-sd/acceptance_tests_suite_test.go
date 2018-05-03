package acceptance_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	helpers_config "github.com/cloudfoundry-incubator/cf-test-helpers/config"
	"github.com/onsi/gomega/gexec"
)

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Acceptance Suite")
}

const Timeout_Short = 10 * time.Second

var (
	allDeployedInstances []instanceInfo
	config               *helpers_config.Config
	appsDir              string
)

var _ = BeforeSuite(func() {
	config = helpers_config.LoadConfig()

	Expect(cf.Cf("api", "--skip-ssl-validation", config.ApiEndpoint).Wait(Timeout_Short)).To(gexec.Exit(0))
	AuthAsAdmin()

	appsDir = os.Getenv("APPS_DIR")
	Expect(appsDir).NotTo(BeEmpty())
})

type instanceInfo struct {
	IP            string
	InstanceID    string
	InstanceGroup string
	Index         string
}

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

func createAndTargetOrgAndSpace(orgName, spaceName string) {
	Expect(cf.Cf("create-org", orgName).Wait(Timeout_Cf)).To(gexec.Exit(0))
	Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Cf)).To(gexec.Exit(0))

	Expect(cf.Cf("create-space", spaceName, "-o", orgName).Wait(Timeout_Cf)).To(gexec.Exit(0))
	Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Cf)).To(gexec.Exit(0))
}

func getAppGUID(appName string) string {
	session := cf.Cf("app", appName, "--guid").Wait(10 * time.Second)
	return string(session.Out.Contents())
}

func getInternalIP(appName string, index int) string {
	session := cf.Cf("ssh", appName, "-i", strconv.Itoa(index), "-c", "echo $CF_INSTANCE_INTERNAL_IP").Wait(10 * time.Second)
	return strings.TrimSpace(string(session.Out.Contents()))
}

func digForNumberOfIPs(hostName string, expectedLength int) []string {
	proxyIPs := []string{}
	Eventually(func() []string {
		resp, err := http.Get(hostName)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		ipsJson, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())

		err = json.Unmarshal(bytes.TrimSpace(ipsJson), &proxyIPs)
		Expect(err).NotTo(HaveOccurred())

		return proxyIPs
	}, 5*time.Second).Should(HaveLen(expectedLength))
	return proxyIPs
}

func scaleApp(appName string, instances int) {
	Expect(cf.Cf(
		"scale", appName,
		"-i", strconv.Itoa(instances),
	).Wait(Timeout_Cf)).To(gexec.Exit(0))
}

func appDir(appType string) string {
	return filepath.Join(appsDir, appType)
}

func pushApp(appName string, instances int) {
	Expect(cf.Cf(
		"push", appName,
		"-p", appDir("proxy"),
		"-f", defaultManifest("proxy"),
		"-i", strconv.Itoa(instances),
	).Wait(Timeout_Cf)).To(gexec.Exit(0))
}

func defaultManifest(appType string) string {
	return filepath.Join(appDir(appType), "manifest.yml")
}
