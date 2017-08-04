package acceptance_test

import (
	"cf-pusher/cf_cli_adapter"
	"fmt"
	"math/rand"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("ASGs and Overlay Policy interaction", func() {
	var (
		cli     *cf_cli_adapter.Adapter
		orgName string
	)

	BeforeEach(func() {
		cli = &cf_cli_adapter.Adapter{CfCliPath: "cf"}
	})

	AfterEach(func() {
		By("deleting the org")
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	Context("when a wide open ASG is configured", func() {
		var (
			asgName      string
			appProxy     string
			appSmoke     string
			appInstances int
			spaceName    string
		)

		BeforeEach(func() {
			appInstances = testConfig.AppInstances
			appProxy = fmt.Sprintf("%s-%s-%d", testConfig.Prefix, "proxy", rand.Int31())
			appSmoke = fmt.Sprintf("%s-%s-%d", testConfig.Prefix, "smoke", rand.Int31())
			asgName = fmt.Sprintf("wide-open-asg-%d", rand.Int31())

			By("creating the org and space")
			orgName = testConfig.Prefix + "wide-open-interaction-org"
			spaceName = testConfig.Prefix + "wide-open-interaction-space"
			setupOrgAndSpace(orgName, spaceName)

			By("creating and binding a wide open security group")
			createASG(cli, asgName, wideOpenASG)
			Expect(cli.BindSecurityGroup(asgName, orgName, spaceName)).To(Succeed())
		})

		AfterEach(func() {
			By("deleting the security group")
			removeASG(cli, asgName)
		})

		It("does not allow traffic on the overlay network without policies", func() {
			By("pushing the proxy and smoke test apps")
			pushApp(appProxy, "proxy")
			pushApp(appSmoke, "smoke", "--no-start")
			setEnv(appSmoke, "PROXY_APP_URL", fmt.Sprintf("http://%s.%s", appProxy, config.AppsDomain))
			start(appSmoke)

			scaleApp(appSmoke, appInstances)

			appsSmoke := []string{appSmoke}

			By(fmt.Sprintf("checking that %s can NOT reach %s", appProxy, appsSmoke))
			assertSelfProxyConnectionFails(appSmoke, appInstances)
		})
	})

	Context("when overlay policies are in place", func() {
		var (
			appProxy  string
			spaceName string
		)

		BeforeEach(func() {
			By("creating the org and space")
			appProxy = fmt.Sprintf("%s-%s-%d", testConfig.Prefix, "proxy", rand.Int31())
			orgName = testConfig.Prefix + "overlay-interaction-org"
			spaceName = testConfig.Prefix + "overlay-interaction-space"
			setupOrgAndSpace(orgName, spaceName)

			By("unbinding all running ASGs")
			for _, sg := range testConfig.DefaultSecurityGroups {
				Expect(cf.Cf("unbind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
			}

			By("pushing the test app")
			pushProxy(appProxy)
		})

		AfterEach(func() {
			By("adding back all the original running ASGs")
			for _, sg := range testConfig.DefaultSecurityGroups {
				Expect(cf.Cf("bind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
			}
		})

		It("continues to enforce ASGs default deny", func() {
			By("creating a policy")
			err := cli.AllowAccess(appProxy, appProxy, 7777, "tcp")
			Expect(err).NotTo(HaveOccurred())

			By("checking that default deny is still enforced")
			assertResponseContains(fmt.Sprintf("%s.%s", appProxy, config.AppsDomain), 80, appProxy, "request failed")
		})
	})
})

func assertSelfProxyConnectionFails(sourceApp string, appInstances int) {
	for i := 0; i < appInstances; i++ {
		assertSelfProxyResponseContains(sourceApp, "FAILED")
	}
}

func assertSelfProxyResponseContains(sourceAppName, desiredResponse string) {
	proxyTest := func() (string, error) {
		resp, err := httpGetBytes(fmt.Sprintf("http://%s.%s/selfproxy", sourceAppName, config.AppsDomain))
		if err != nil {
			return "", err
		}
		return string(resp.Body), nil
	}
	Eventually(proxyTest, 10*time.Second, 500*time.Millisecond).Should(ContainSubstring(desiredResponse))
}

func pushApp(appName, kind string, extraArgs ...string) {
	args := append([]string{
		"push", appName,
		"-p", appDir(kind),
		"-f", defaultManifest(kind),
	}, extraArgs...)
	Expect(cf.Cf(args...).Wait(Timeout_Push)).To(gexec.Exit(0))
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

var wideOpenASG string = `
[
		{
			"destination": "0.0.0.0/0",
			"protocol": "all"
		}
	]
`
