package acceptance_test

import (
	"cf-pusher/cf_cli_adapter"
	"fmt"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("ASGs and Overlay Policy interaction", func() {
	var (
		appProxy     string
		appSmoke     string
		appInstances int
		prefix       string
		spaceName    string
		orgName      string
		asgName      string
		cli          *cf_cli_adapter.Adapter
	)

	BeforeEach(func() {
		prefix = testConfig.Prefix

		orgName = prefix + "interaction-org"
		Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

		spaceName = prefix + "interaction-space"
		Expect(cf.Cf("create-space", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))

		appInstances = testConfig.AppInstances

		appProxy = prefix + "proxy"
		appSmoke = prefix + "smoke"

		cli = &cf_cli_adapter.Adapter{CfCliPath: "cf"}
		asgName = "wide-open-asg"

		By("creating and binding a wide open security group")
		createASG(cli, asgName, wideOpenASG)
		Expect(cli.BindSecurityGroup(asgName, orgName, spaceName)).To(Succeed())
	})

	AfterEach(func() {
		By("deleting the org")
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
		By("deleting the security group")
		removeASG(cli, asgName)
	})

	It("does not allow traffic on the overlay network without policies", func(done Done) {
		By("pushing the proxy and smoke test apps")
		pushApp(appProxy, "proxy")
		pushApp(appSmoke, "smoke", "--no-start")
		setEnv(appSmoke, "PROXY_APP_URL", fmt.Sprintf("http://%s.%s", appProxy, config.AppsDomain))
		start(appSmoke)

		scaleApp(appSmoke, appInstances)

		appsSmoke := []string{appSmoke}

		By(fmt.Sprintf("checking that %s can NOT reach %s", appProxy, appsSmoke))
		assertSelfProxyConnectionFails(appSmoke, appInstances)

		close(done)
	}, 30*60 /* <-- overall spec timeout in seconds */)
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
