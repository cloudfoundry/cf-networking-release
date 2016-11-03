package netman_cf_upgrade_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("apps remain available during an upgrade deploy", func() {
	var (
		NoASGTargetIP string
		ASGTargetIP   string
		ASGFilepath   string
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

		ASGTargetIP = boshIPFor("router")
		NoASGTargetIP = boshIPFor("uaa")
		By(fmt.Sprintf("found ASG Target IPs (allow %s) (deny %s)", ASGTargetIP, NoASGTargetIP))

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

		By("pushing the proxy app")
		Expect(cli.Push("proxy-upgrade", "../example-apps/proxy", "../example-apps/proxy/manifest.yml")).To(Succeed())
		Expect(cli.Scale("proxy-upgrade", 3)).To(Succeed())

		By("checking the app has started")
		Eventually(checkStatusCode).Should(Equal(http.StatusOK))

		By("checking the app continuously")
		var failures []string
		go checkStatusCodeContinuously(ASGTargetIP, NoASGTargetIP, &failures)

		By("deploying upgrade manifest")
		boshDeploy(upgradeManifest)
		fmt.Printf("\n\n### Got %d failures ###\n\n", len(failures))
		fmt.Println(strings.Join(failures, "\n"))
		Expect(len(failures)).To(BeNumerically("<", 5))
	})
})

func checkASG(ip string) (int, string) {
	resp, err := http.Get(fmt.Sprintf("http://proxy-upgrade.%s/proxy/%s", config.AppsDomain, ip))
	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return http.StatusTeapot, string(dump)
	}
	defer resp.Body.Close()
	return resp.StatusCode, string(dump)
}

func checkApp() (int, string) {
	resp, err := http.Get("http://proxy-upgrade." + config.AppsDomain)
	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return http.StatusTeapot, string(dump)
	}
	defer resp.Body.Close()
	return resp.StatusCode, string(dump)
}

func checkStatusCode() int {
	resp, err := http.Get("http://proxy-upgrade." + config.AppsDomain)
	if err != nil {
		return http.StatusTeapot
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func checkStatusCodeContinuously(allowIP, denyIP string, failures *[]string) {
	defer GinkgoRecover()
	for {
		sc, dump := checkApp()
		if sc != http.StatusOK {
			*failures = append(*failures, dump)
		}
		sc, dump = checkASG(allowIP)
		if sc != http.StatusOK {
			*failures = append(*failures, dump)
		}
		sc, dump = checkASG(denyIP)
		if sc != http.StatusInternalServerError {
			*failures = append(*failures, dump)
		}
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
