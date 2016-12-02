package acceptance_test

import (
	"cf-pusher/cf_cli_adapter"
	"fmt"
	"lib/testsupport"
	"math/rand"
	"os"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("policy cleanup", func() {
	var (
		appA, appB  string
		orgName     string
		spaceName   string
		cli         *cf_cli_adapter.Adapter
		ASGFilepath string
	)

	BeforeEach(func() {
		appA = fmt.Sprintf("appA-%d", rand.Int31())
		appB = fmt.Sprintf("appB-%d", rand.Int31())

		cli = &cf_cli_adapter.Adapter{
			CfCliPath: "cf",
		}
		AuthAsAdmin()

		orgName = "asg-org"
		Expect(cli.CreateOrg(orgName)).To(Succeed())
		Expect(cli.TargetOrg(orgName)).To(Succeed())

		spaceName = "asg-space"
		Expect(cli.CreateSpace(spaceName)).To(Succeed())
		Expect(cli.TargetSpace(spaceName)).To(Succeed())

	})

	AfterEach(func() {
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cli.DeleteSecurityGroup("big-asg")).To(Succeed())
		os.Remove(ASGFilepath)
	})

	Describe("Pushing app with and without ASG", func() {
		It("should not give large time difference", func() {
			By("pushing an app")
			start := time.Now()
			pushProxy(appB)
			duration := time.Since(start)

			By("creating large ASG")
			asg := testsupport.BuildASG(1000)
			var err error
			ASGFilepath, err = testsupport.CreateASGFile(asg)
			Expect(err).NotTo(HaveOccurred())
			Expect(cli.CreateSecurityGroup("big-asg", ASGFilepath)).To(Succeed())
			By("binding ASG to the space")
			Expect(cli.BindSecurityGroup("big-asg", orgName, spaceName)).To(Succeed())

			By("pushing an app")
			start = time.Now()
			pushProxy(appA)
			durationWithASG := time.Since(start)

			fmt.Println("##############################################")
			fmt.Println("push app without ASG took:", duration)
			fmt.Println("push app with big ASG took:", durationWithASG)
			fmt.Println("##############################################")
		})
	})
})
