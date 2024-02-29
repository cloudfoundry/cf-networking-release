package acceptance_test

import (
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/lib/testsupport"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Container startup time with a big ASG", func() {
	var (
		orgName     string
		spaceName   string
		ASGFilepath string
	)

	BeforeEach(func() {
		AuthAsAdmin()

		orgName = "asg-org"
		Expect(cfCLI.CreateOrg(orgName)).To(Succeed())
		Expect(cfCLI.TargetOrg(orgName)).To(Succeed())

		spaceName = "asg-space"
		Expect(cfCLI.CreateSpace(spaceName, orgName)).To(Succeed())
		Expect(cfCLI.TargetSpace(spaceName)).To(Succeed())
	})

	AfterEach(func() {
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cfCLI.DeleteSecurityGroup("big-asg")).To(Succeed())
		os.Remove(ASGFilepath)
	})

	Describe("Pushing app with and without a large application security group (ASG)", func() {
		It("should not give large time difference", func() {
			var (
				durationWithoutASG time.Duration
				durationWithASG    time.Duration
			)

			By("pushing an app", func() {
				start := time.Now()
				appB := fmt.Sprintf("appB-%d", randomGenerator.Int31())
				pushProxy(appB)
				durationWithoutASG = time.Since(start)
			})

			By("creating a large ASG", func() {
				asg := testsupport.BuildASG(1000)
				var err error
				ASGFilepath, err = testsupport.CreateTempFile(asg)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfCLI.CreateSecurityGroup("big-asg", ASGFilepath)).To(Succeed())
				Expect(cfCLI.BindSecurityGroup("big-asg", orgName, spaceName)).To(Succeed())
			})

			By("pushing another app", func() {
				start := time.Now()
				appA := fmt.Sprintf("appA-%d", randomGenerator.Int31())
				pushProxy(appA)
				durationWithASG = time.Since(start)
			})

			fmt.Println("##############################################")
			fmt.Println("push app without ASG took:", durationWithoutASG)
			fmt.Println("push app with big ASG took:", durationWithASG)
			fmt.Println("##############################################")
		})
	})
})
