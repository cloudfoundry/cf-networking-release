package acceptance_test

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("c2c traffic source ip", func() {
	var (
		appName string
		apps    []AppInstance
	)

	BeforeEach(func() {
		appName = fmt.Sprintf("appA-%d", rand.Int31())

		appCount := 5
		By("pushing the test app")
		pushAppWithInstanceCount(appName, appCount)
		apps = getAppInstances(appName, appCount)

		By("adding a network policy")
		Expect(cfCLI.AddNetworkPolicy(appName, appName, 8080, "tcp")).To(Succeed())

		By(fmt.Sprintf("waiting %s for policies to be created on cells", time.Duration(PolicyWaitTime)))
		time.Sleep(PolicyWaitTime)
	})

	It("should be the container's ip", func() {
		By("checking when the apps instances are on the same host")
		app1, app2 := findTwoInstancesOnTheSameHost(apps)
		app2Curl := fmt.Sprintf("curl --fail http://%s:8080/echosourceip", app2.internalIP)

		session := cf.Cf("ssh", appName, "-i", app1.index, "-c", app2Curl)
		Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(string(session.Out.Contents())).To(ContainSubstring(app1.internalIP))

		By("checking when the apps instances are on different same hosts")
		app1, app2 = findTwoInstancesOnDifferentHosts(apps)
		app2Curl = fmt.Sprintf("curl --fail http://%s:8080/echosourceip", app2.internalIP)

		session = cf.Cf("ssh", appName, "-i", app1.index, "-c", app2Curl)
		Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(string(session.Out.Contents())).To(ContainSubstring(app1.internalIP))
	})
})
