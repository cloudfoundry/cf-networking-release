package netman_cf_upgrade_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("apps remain available during an upgrade deploy", func() {
	var (
		ASGTargetIP string
		ASGFilepath string
	)

	AfterEach(func() {
		os.Remove(ASGFilepath)
	})

	It("upgrades CF with no downtime", func() {
		org, space := "upgrade-org", "upgrade-space"

		baseManifest := os.Getenv("BASE_MANIFEST")
		upgradeManifest := os.Getenv("UPGRADE_MANIFEST")
		By("deleting the deployment")
		boshDeleteDeployment()

		By("deploying base manifest")
		boshDeploy(baseManifest)

		By("finding the ASGTargetIP")
		ASGTargetIP = boshIPFor("router")

		By("pushing the proxy app")
		Expect(cli.SetApiWithoutSsl(config.ApiEndpoint)).To(Succeed())
		Expect(cli.Auth(config.AdminUser, config.AdminPassword)).To(Succeed())
		Expect(cli.CreateOrg(org)).To(Succeed())
		Expect(cli.TargetOrg(org)).To(Succeed())
		Expect(cli.CreateSpace(space)).To(Succeed())
		Expect(cli.TargetSpace(space)).To(Succeed())

		By("create and bind security group")

		asg := `[
		 {
		 "protocol": "tcp",
		 "destination": "` + ASGTargetIP + `",
		 "ports": "80"
		 }
		 ]
		 `
		ASGFilepath = createASGFile(asg)
		Expect(cli.CreateSecurityGroup("test-running-asg", ASGFilepath)).To(Succeed())
		Expect(cli.BindSecurityGroup("test-running-asg", org, space)).To(Succeed())

		Expect(cli.Push("proxy-upgrade", "../example-apps/proxy", "../example-apps/proxy/manifest.yml")).To(Succeed())
		Expect(cli.Scale("proxy-upgrade", 3)).To(Succeed())

		By("checking the app has started")
		Eventually(checkStatusCode).Should(Equal(http.StatusOK))

		By("checking the app continuously")
		go checkStatusCodeContinuously(ASGTargetIP)

		By("deploying upgrade manifest")
		boshDeploy(upgradeManifest)
	})
})

func checkASG(ip string) int {
	resp, err := http.Get(fmt.Sprintf("http://proxy-upgrade.%s/proxy/%s", config.AppsDomain, ip))
	if err != nil {
		return http.StatusTeapot
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func checkStatusCode() int {
	resp, err := http.Get("http://proxy-upgrade." + config.AppsDomain)
	if err != nil {
		return http.StatusTeapot
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func checkStatusCodeContinuously(ip string) {
	defer GinkgoRecover()
	for {
		Expect(checkASG(ip)).To(Equal(http.StatusOK))
		Expect(checkStatusCode()).To(Equal(http.StatusOK))
		time.Sleep(1 * time.Second)
	}
}

func createASGFile(asg string) string {
	asgFile, err := ioutil.TempFile("", "")
	Expect(err).NotTo(HaveOccurred())
	path := asgFile.Name()
	Expect(ioutil.WriteFile(path, []byte(asg), os.ModePerm))
	return path
}
