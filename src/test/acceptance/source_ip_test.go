package acceptance_test

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"

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
		apps := pushApps(appName, APP_COUNT)
		var err error
		app1, app2, err = findTwoInstancesOnTheSameHost(apps)
		Expect(err).NotTo(HaveOccurred())

		By("adding a network policy")
		Expect(cf.Cf("add-network-policy", appName, "--destination-app", appName).Wait(Timeout_Push)).To(gexec.Exit(0))
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

func pushApps(appName string, appCount int) []AppInstance {
	pushProxy(appName)
	scaleApp(appName, appCount)

	apps := make([]AppInstance, appCount)
	for i := 0; i < appCount; i++ {
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

func findTwoInstancesOnTheSameHost(apps []AppInstance) (AppInstance, AppInstance, error) {
	hostsToApps := map[string]AppInstance{}

	for _, app := range apps {
		foundApp, ok := hostsToApps[app.hostIdentifier]
		if ok {
			return foundApp, app, nil
		}
		hostsToApps[app.hostIdentifier] = app
	}

	return AppInstance{}, AppInstance{}, errors.New("Failed to find two instances on the same host")
}
