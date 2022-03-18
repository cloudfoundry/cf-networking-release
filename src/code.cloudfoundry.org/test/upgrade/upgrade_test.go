package upgrade_test

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

		ASGTargetIP := boshIPFor("router")
		noASGTargetIP := boshIPFor("uaa")
		By(fmt.Sprintf("found ASG Target IPs (allow %s) (deny %s)", ASGTargetIP, noASGTargetIP))

		Expect(cli.SetApiWithoutSsl(conf.ApiEndpoint)).To(Succeed())
		Expect(cli.Auth(conf.AdminUser, conf.AdminPassword)).To(Succeed())
		Expect(cli.CreateOrg(org)).To(Succeed())
		Expect(cli.TargetOrg(org)).To(Succeed())
		Expect(cli.CreateSpace(space, org)).To(Succeed())
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
		var appFailures []string
		var ASGFailures []string
		var noASGFailures []string
		go checkContinuously("http://proxy-upgrade."+conf.AppsDomain, http.StatusOK, &appFailures)
		go checkContinuously(fmt.Sprintf("http://proxy-upgrade.%s/proxy/%s", conf.AppsDomain, ASGTargetIP), http.StatusOK, &ASGFailures)
		go checkContinuously(fmt.Sprintf("http://proxy-upgrade.%s/proxy/%s", conf.AppsDomain, noASGTargetIP), http.StatusInternalServerError, &noASGFailures)

		By("deploying upgrade manifest")
		boshDeploy(upgradeManifest)

		fmt.Printf("\n\n### Got %d app failures ###\n\n", len(appFailures))
		fmt.Println(strings.Join(appFailures, "\n"))
		fmt.Printf("\n\n### Got %d ASG failures ###\n\n", len(ASGFailures))
		fmt.Println(strings.Join(ASGFailures, "\n"))
		fmt.Printf("\n\n### Got %d no ASG failures ###\n\n", len(noASGFailures))
		fmt.Println(strings.Join(noASGFailures, "\n"))

		Expect(len(appFailures)).To(BeNumerically("<", 5))
		Expect(len(ASGFailures)).To(BeNumerically("<", 5))
		Expect(len(noASGFailures)).To(BeNumerically("<", 5))

		By("deleting the deployment")
		boshDeleteDeployment()
	})
})

func check(url string) (int, string) {
	resp, err := http.Get(url)
	if err != nil {
		return http.StatusTeapot, ""
	}
	defer resp.Body.Close()

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return http.StatusTeapot, string(dump)
	}

	return resp.StatusCode, string(dump)
}

func checkStatusCode() int {
	sc, _ := check("http://proxy-upgrade." + conf.AppsDomain)
	return sc
}

func checkContinuously(url string, statusCode int, failures *[]string) {
	defer GinkgoRecover()
	for {
		sc, dump := check(url)
		if sc != statusCode {
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
