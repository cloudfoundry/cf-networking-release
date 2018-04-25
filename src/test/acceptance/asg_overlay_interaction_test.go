package acceptance_test

import (
	"cf-pusher/cf_cli_adapter"
	"errors"
	"fmt"
	"math/rand"
	"strings"
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
			spaceName    string
			appInstances []AppInstance
		)

		BeforeEach(func() {
			appProxy = fmt.Sprintf("%s-%s-%d", testConfig.Prefix, "proxy", rand.Int31())
			asgName = fmt.Sprintf("wide-open-asg-%d", rand.Int31())

			By("creating the org and space")
			orgName = testConfig.Prefix + "wide-open-interaction-org"
			spaceName = testConfig.Prefix + "wide-open-interaction-space"
			setupOrgAndSpace(orgName, spaceName)

			By("pushing proxy app with 5 instances")
			pushApps(appProxy, 5)

			By("create a wide open ASG")
			createASG(cli, asgName, wideOpenASG)
			Expect(cli.BindSecurityGroup(asgName, orgName, spaceName)).To(Succeed())

			By("restage proxy app")
			restage(appProxy)
			appInstances = getAppInstances(appProxy, 5)
		})

		AfterEach(func() {
			By("deleting the security group")
			removeASG(cli, asgName)
		})

		Context("when no policies are added", func() {
			It("does not allow traffic on the overlay network", func() {
				By("checking connectivity fails between two instances on the same cell")
				app1, app2 := findTwoInstancesOnTheSameHost(appInstances)

				app2Curl := fmt.Sprintf("curl --fail http://%s:8080/echosourceip", app2.internalIP)
				session := cf.Cf("ssh", appProxy, "-i", app1.index, "-c", app2Curl)
				Expect(session.Wait(Timeout_Push)).ToNot(gexec.Exit(0))

				By("checking connectivity fails between two instances on the different cells")
				app1, app2 = findTwoInstancesOnTheDifferentHost(appInstances)

				app2Curl = fmt.Sprintf("curl --fail http://%s:8080/echosourceip", app2.internalIP)
				session = cf.Cf("ssh", appProxy, "-i", app1.index, "-c", app2Curl)
				Expect(session.Wait(Timeout_Push)).ToNot(gexec.Exit(0))
			})
		})

		Context("when a policy is added", func() {
			BeforeEach(func() {
				By("creating a policy")
				err := cli.AddNetworkPolicy(appProxy, appProxy, 8080, "tcp")
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("waiting %s for policies to be created on cells", time.Duration(PolicyWaitTime)))
				time.Sleep(PolicyWaitTime)
			})

			It("does allow traffic on the overlay network", func() {
				By("checking connectivity fails between two instances on the same cell")
				app1, app2 := findTwoInstancesOnTheSameHost(appInstances)

				app2Curl := fmt.Sprintf("curl --fail http://%s:8080/echosourceip", app2.internalIP)
				session := cf.Cf("ssh", appProxy, "-i", app1.index, "-c", app2Curl)
				Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))

				By("checking connectivity fails between two instances on the different cells")
				app1, app2 = findTwoInstancesOnTheDifferentHost(appInstances)

				app2Curl = fmt.Sprintf("curl --fail http://%s:8080/echosourceip", app2.internalIP)
				session = cf.Cf("ssh", appProxy, "-i", app1.index, "-c", app2Curl)
				Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))
			})
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
				Expect(cf.Cf("unbind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit())
			}

			By("pushing the test app")
			pushProxy(appProxy)
		})

		AfterEach(func() {
			By("adding back all the original running ASGs")
			for _, sg := range testConfig.DefaultSecurityGroups {
				Expect(cf.Cf("bind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit())
			}
		})

		It("continues to enforce ASGs default deny", func() {
			By("creating a policy")
			err := cli.AddNetworkPolicy(appProxy, appProxy, 7777, "tcp")
			Expect(err).NotTo(HaveOccurred())

			By("checking that default deny is still enforced")
			assertResponseContains(fmt.Sprintf("%s.%s", appProxy, config.AppsDomain), 80, appProxy, "request failed")
		})
	})
})

func assertSelfProxyConnectionFails(sourceApp string, appInstances int) {
	for i := 0; i < appInstances; i++ {
		assertSelfProxyResponseContains(sourceApp, i, "FAILED")
	}
}

func assertSelfProxyResponseContains(sourceAppName string, index int, desiredResponse string) {
	proxyTest := func() (string, error) {
		session := cf.Cf("ssh", sourceAppName, "-i", fmt.Sprintf("%d", index), "-c", "curl --silent http://localhost:8080/selfproxy").Wait(Timeout_Push)
		if session.ExitCode() != 0 {
			return "", fmt.Errorf("proxy test exit code: %s\n%s", session.ExitCode(), string(session.Err.Contents()))
		}

		return string(session.Out.Contents()), nil
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

func restage(appName string) {
	Expect(cf.Cf(
		"restage", appName,
	).Wait(Timeout_Push)).To(gexec.Exit(0))
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

func findTwoInstancesOnTheDifferentHost(apps []AppInstance) (AppInstance, AppInstance) {
	for _, app := range apps[1:] {
		if apps[0].hostIdentifier != app.hostIdentifier {
			return apps[0], app
		}
	}

	Expect(errors.New("Failed to find two instances on different host")).ToNot(HaveOccurred())
	return AppInstance{}, AppInstance{}
}

var wideOpenASG string = `
[
		{
			"destination": "0.0.0.0/0",
			"protocol": "all"
		}
	]
`
