package smoke_test

import (
	"path/filepath"
	"time"

	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const Timeout_Cf = 4 * time.Minute

var (
	prefix       string
	orgName      string
	appName      string
	queryAppName string
)

var _ = Describe("Push App Smoke", func() {

	BeforeEach(func() {
		prefix = config.Prefix

		if config.SmokeOrg == "" {
			orgName = prefix + "org" // cf-pusher expects this name
			Expect(cf.Cf("create-org", orgName).Wait(Timeout_Cf)).To(gexec.Exit(0))
		} else {
			orgName = config.SmokeOrg
		}
		Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Cf)).To(gexec.Exit(0))

		spaceName := config.SmokeSpace
		if spaceName == "" {
			spaceName = prefix + "space" // cf-pusher expects this name
			Expect(cf.Cf("create-space", spaceName, "-o", orgName).Wait(Timeout_Cf)).To(gexec.Exit(0))
		}
		Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Cf)).To(gexec.Exit(0))

		appName = prefix + "proxy"
		queryAppName = prefix + "queryProxy"

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

	Describe("when performing a dns lookup for a domain configured to point to the bosh adapter", func() {
		Measure("resolves its infrastructure name within 5 seconds after push", func(b Benchmarker) {
			By("pushing the app")
			b.Time("push", func() {
				pushProxyWithInternalRoute(appName)
			})

			By("getting an answer in the dig response within 5 seconds of app push finishing")
			proxyIPs := []string{}

			httpClient := NewClient()
			b.Time("digAnswer", func() {
				Eventually(func() []string {
					resp, err := httpClient.Get("http://" + queryAppName + "." + config.AppsDomain + "/dig/app-smoke.apps.internal")
					if err != nil || resp.StatusCode != http.StatusOK {
						return []string{}
					}

					ipsJson, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())

					err = json.Unmarshal(bytes.TrimSpace(ipsJson), &proxyIPs)
					Expect(err).NotTo(HaveOccurred())

					return proxyIPs
				}, 5*time.Second).Should(HaveLen(1))
			})

			By("asserting that the answer equals the app's internal ip")
			var proxyContainerIp string
			b.Time("ssh", func() {
				session := cf.Cf("ssh", appName, "-c", "echo $CF_INSTANCE_INTERNAL_IP").Wait(10 * time.Second)
				proxyContainerIp = string(session.Out.Contents())
			})

			Expect(proxyIPs).To(ConsistOf(strings.TrimSpace(proxyContainerIp)))
		}, 1)
	})
})

func appDir(appType string) string {
	return filepath.Join(appsDir, appType)
}

func pushProxy(appName string) {
	Expect(cf.Cf(
		"push", appName,
		"-p", appDir("proxy"),
		"-f", defaultManifest("proxy"),
	).Wait(Timeout_Cf)).To(gexec.Exit(0))
}

func pushProxyWithInternalRoute(appName string) {
	Expect(cf.Cf(
		"push", appName,
		"-p", appDir("proxy"),
		"-f", internalRouteManifest("proxy"),
	).Wait(Timeout_Cf)).To(gexec.Exit(0))
}

func defaultManifest(appType string) string {
	return filepath.Join(appDir(appType), "manifest.yml")
}

func internalRouteManifest(appType string) string {
	return filepath.Join(appDir(appType), "internal-route-manifest.yml")
}
