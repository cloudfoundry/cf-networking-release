package netman_cf_upgrade_test

import (
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("apps remain available during an upgrade deploy", func() {
	It("upgrades CF with no downtime", func() {
		baseManifest := os.Getenv("BASE_MANIFEST")
		upgradeManifest := os.Getenv("UPGRADE_MANIFEST")
		By("deleting the deployment")
		boshDeleteDeployment()

		By("deploying base manifest")
		boshDeploy(baseManifest)

		By("pushing the proxy app")
		Expect(cli.SetApiWithoutSsl(config.ApiEndpoint)).To(Succeed())
		Expect(cli.Auth(config.AdminUser, config.AdminPassword)).To(Succeed())
		Expect(cli.CreateOrg("upgrade-org")).To(Succeed())
		Expect(cli.TargetOrg("upgrade-org")).To(Succeed())
		Expect(cli.CreateSpace("upgrade-space")).To(Succeed())
		Expect(cli.TargetSpace("upgrade-space")).To(Succeed())
		Expect(cli.Push("proxy-upgrade", "../example-apps/proxy", "../example-apps/proxy/manifest.yml")).To(Succeed())
		Expect(cli.Scale("proxy-upgrade", 3)).To(Succeed())

		By("checking the app has started")
		Eventually(checkStatusCode).Should(Equal(http.StatusOK))

		By("checking the app continuously")
		go checkStatusCodeContinuously()

		By("deploying upgrade manifest")
		boshDeploy(upgradeManifest)
	})
})

func checkStatusCode() int {
	resp, err := http.Get("http://proxy-upgrade." + config.AppsDomain)
	if err != nil {
		return http.StatusTeapot
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func checkStatusCodeContinuously() {
	defer GinkgoRecover()
	for {
		Expect(checkStatusCode()).To(Equal(http.StatusOK))
		time.Sleep(1 * time.Second)
	}
}
