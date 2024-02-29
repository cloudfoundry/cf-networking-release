package smoke_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Service Discovery Smoke", func() {
	var (
		prefix       string
		orgName      string
		appName      string
		queryAppName string
	)

	BeforeEach(func() {
		prefix = config.Prefix
		appName = prefix + "proxy"
		queryAppName = prefix + "queryProxy"

		if config.SmokeOrg == "" {
			orgName = prefix + "org"
			Expect(cf.Cf("create-org", orgName).Wait(Timeout_Cf)).To(gexec.Exit(0))
		} else {
			orgName = config.SmokeOrg
		}
		Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Cf)).To(gexec.Exit(0))

		spaceName := config.SmokeSpace
		if spaceName == "" {
			spaceName = prefix + "space"
			Expect(cf.Cf("create-space", spaceName, "-o", orgName).Wait(Timeout_Cf)).To(gexec.Exit(0))
		}
		Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Cf)).To(gexec.Exit(0))

		By("pushing query app")
		pushProxy(queryAppName)
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(Timeout_Cf))
		Expect(cf.Cf("delete", queryAppName, "-f").Wait(Timeout_Cf))

		if config.SmokeOrg == "" {
			Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Cf)).To(gexec.Exit(0))
		}
	})

	It("resolves an internal route from an app", func() {
		By("pushing the app")
		pushProxyWithInternalRoute(appName)

		By("asserting that the answer equals the app's internal ip")
		var proxyContainerIp string
		session := cf.Cf("ssh", appName, "-c", "echo $CF_INSTANCE_INTERNAL_IP").Wait(10 * time.Second)
		proxyContainerIp = string(session.Out.Contents())

		Eventually(func() ([]string, error) {
			_, ips, err := digFromApp(queryAppName + "." + config.AppsDomain)
			return ips, err
		}, 20*time.Second).Should(ConsistOf(strings.TrimSpace(proxyContainerIp)))

		By("deleting the app")
		deleteProxy(appName)

		Eventually(func() (int, error) {
			statusCode, _, err := digFromApp(queryAppName + "." + config.AppsDomain)
			return statusCode, err
		}, 20*time.Second).Should(Equal(http.StatusInternalServerError))
	})
})

func pushProxy(appName string) {
	Expect(cf.Cf(
		"push", appName,
		"-p", appDir("proxy"),
		"-f", defaultManifest("proxy"),
	).Wait(Timeout_Cf)).To(gexec.Exit(0))
}

func deleteProxy(appName string) {
	Expect(cf.Cf(
		"delete", appName, "-f",
	).Wait(Timeout_Cf)).To(gexec.Exit(0))
}

func pushProxyWithInternalRoute(appName string) {
	Expect(cf.Cf(
		"push", appName,
		"-p", appDir("proxy"),
		"-f", internalRouteManifest("proxy"),
	).Wait(Timeout_Cf)).To(gexec.Exit(0))
}

func digFromApp(queryAppRoute string) (int, []string, error) {
	proxyIPs := []string{}
	httpClient := NewClient()

	resp, err := httpClient.Get("http://" + queryAppRoute + "/dig/app-smoke.apps.internal")
	if err != nil {
		return 0, []string{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, []string{}, nil
	}

	ipsJson, err := io.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())

	err = json.Unmarshal(bytes.TrimSpace(ipsJson), &proxyIPs)
	Expect(err).NotTo(HaveOccurred())

	return resp.StatusCode, proxyIPs, nil
}

func appDir(appType string) string {
	return filepath.Join(appsDir, appType)
}

func defaultManifest(appType string) string {
	return filepath.Join(appDir(appType), "manifest.yml")
}

func internalRouteManifest(appType string) string {
	return filepath.Join(appDir(appType), "internal-route-manifest.yml")
}
