package acceptance_test

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = FDescribe("ASGs and Overlay Policy interaction", func() {
	var (
		orgName string
	)

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
			orgName = TestSetup.TestSpace.OrganizationName()
			spaceName = TestSetup.TestSpace.SpaceName()

			appCount := 5
			By(fmt.Sprintf("pushing proxy app with %d instances", appCount))
			pushAppWithInstanceCount(appProxy, appCount)

			By("create a wide open ASG")
			createASG(asgName, `[{"destination":"0.0.0.0/0","protocol":"all"}]`)
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Timeout_Push, func() {
				Expect(cfCLI.BindSecurityGroup(asgName, orgName, spaceName)).To(Succeed())
			})

			By("restage proxy app")
			restage(appProxy)
			appInstances = getAppInstances(appProxy, appCount)
		})

		AfterEach(func() {
			By("deleting the security group")
			removeASG(asgName)
		})

		Context("when no policies are added", func() {
			It("does not allow traffic on the overlay network", func() {
				By("checking connectivity fails between two instances on the same cell")
				app1, app2 := findTwoInstancesOnTheSameHost(appInstances)

				app2Curl := fmt.Sprintf("curl --fail --connect-timeout 10 http://%s:8080/echosourceip", app2.internalIP)
				session := cf.Cf("ssh", appProxy, "-i", app1.index, "-c", app2Curl)
				Expect(session.Wait(Timeout_Push)).ToNot(gexec.Exit(0))

				By("checking connectivity fails between two instances on the different cells")
				app1, app2 = findTwoInstancesOnDifferentHosts(appInstances)

				app2Curl = fmt.Sprintf("curl --fail --connect-timeout 10 http://%s:8080/echosourceip", app2.internalIP)
				session = cf.Cf("ssh", appProxy, "-i", app1.index, "-c", app2Curl)
				Expect(session.Wait(Timeout_Push)).ToNot(gexec.Exit(0))
			})
		})

		Context("when a policy is added", func() {
			BeforeEach(func() {
				By("creating a policy")
				err := cfCLI.AddNetworkPolicy(appProxy, appProxy, 8080, "tcp")
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
				app1, app2 = findTwoInstancesOnDifferentHosts(appInstances)

				app2Curl = fmt.Sprintf("curl --fail http://%s:8080/echosourceip", app2.internalIP)
				session = cf.Cf("ssh", appProxy, "-i", app1.index, "-c", app2Curl)
				Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))
			})
		})
	})

	Context("when overlay policies are in place", func() {
		var (
			appProxy string
		)

		BeforeEach(func() {
			By("creating the org and space")
			appProxy = fmt.Sprintf("%s-%s-%d", testConfig.Prefix, "proxy", rand.Int31())
			orgName = TestSetup.GetOrganizationName()

			By("unbinding all running ASGs")
			for _, sg := range testConfig.DefaultSecurityGroups {
				Expect(cf.Cf("unbind-security-group", sg, TestSetup.TestSpace.OrganizationName(), TestSetup.TestSpace.SpaceName(), "--lifecycle", "running").Wait(Timeout_Short)).To(gexec.Exit(0))
			}

			By("pushing the test app")
			pushProxy(appProxy)
		})

		AfterEach(func() {
			// By("adding back all the original running ASGs")
			// for _, sg := range testConfig.DefaultSecurityGroups {
			// 	Expect(cf.Cf("bind-security-group", sg, TestSetup.TestSpace.OrganizationName(), "--lifecycle", "running", "--space", TestSetup.TestSpace.SpaceName()).Wait(Timeout_Short)).To(gexec.Exit(0))
			// }
		})

		FIt("continues to enforce ASGs default deny", func() {
			By("creating a policy")
			err := cfCLI.AddNetworkPolicy(appProxy, appProxy, 7777, "tcp")
			Expect(err).NotTo(HaveOccurred())

			By("checking that default deny is still enforced")
			assertResponseContains(fmt.Sprintf("%s.%s", appProxy, config.AppsDomain), 80, appProxy, "request failed")
		})
	})
})
