package acceptance_test

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type AppInstance struct {
	hostIdentifier string
	index          string
	internalIP     string
}

const APP_COUNT = 5

var _ = Describe("c2c traffic source ip", func() {
	var (
		appName   string
		orgName   string
		spaceName string
		app1      AppInstance
		app2      AppInstance
	)

	BeforeEach(func() {
		appName = fmt.Sprintf("appA-%d", rand.Int31())

		orgName = testConfig.Prefix + "source-traffic-org"
		spaceName = testConfig.Prefix + "space"
		setupOrgAndSpace(orgName, spaceName)

		By("pushing the test app")
		pushApps(appName, APP_COUNT)
		apps := getAppInstances(appName, APP_COUNT)
		app1, app2 = findTwoInstancesOnTheSameHost(apps)

		By("adding a network policy")
		Expect(cf.Cf("add-network-policy", appName, "--destination-app", appName).Wait(Timeout_Push)).To(gexec.Exit(0))

		By(fmt.Sprintf("waiting %s for policies to be created on cells", time.Duration(PolicyWaitTime)))
		time.Sleep(PolicyWaitTime)
	})

	AfterEach(func() {
		By("deleting the test org")
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	It("should be the container's ip", func() {
		app2Curl := fmt.Sprintf("curl --fail http://%s:8080/echosourceip", app2.internalIP)

		session := cf.Cf("ssh", appName, "-i", app1.index, "-c", app2Curl)
		Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(string(session.Out.Contents())).To(ContainSubstring(app1.internalIP))
	})
})

func pushApps(appName string, appCount int) {
	Expect(cf.Cf(
		"push", appName,
		"-p", appDir("proxy"),
		"-i", fmt.Sprintf("%d", appCount),
		"-f", defaultManifest("proxy"),
	).Wait(Timeout_Push)).To(gexec.Exit(0))
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
	Expect(errors.New("Failed to find two instances on the same host")).ToNot(HaveOccurred())
	return AppInstance{}, AppInstance{}
}
