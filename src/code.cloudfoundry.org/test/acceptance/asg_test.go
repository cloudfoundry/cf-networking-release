package acceptance_test

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Application Security Groups", func() {
	var (
		orgName string
		asgName string
	)

	BeforeEach(func() {
		asgName = fmt.Sprintf("ccAllow-%d", rand.Int31())
	})

	AfterEach(func() {
		By("deleting the asg")
		removeASG(asgName)

		By("adding back all the original running ASGs")
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Timeout_Push, func() {
			for _, sg := range testConfig.DefaultSecurityGroups {
				Expect(cf.Cf("bind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
			}
		})
	})

	var (
		appName   string
		spaceName string
	)

	BeforeEach(func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Timeout_Push, func() {
			By("unbinding all running ASGs")
			for _, sg := range testConfig.DefaultSecurityGroups {
				Expect(cf.Cf("unbind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
			}
		})

		By("creating the org and space")
		orgName = TestSetup.TestSpace.OrganizationName()
		spaceName = TestSetup.TestSpace.SpaceName()

		By("Pushing an app")
		appName = fmt.Sprintf("%s-%s-%d", testConfig.Prefix, "proxy", rand.Int31())
		pushProxy(appName)
	})

	It("applies security group changes", func() {
		internalCCPort := 9024
		proxyRequestURL := fmt.Sprintf("http://%s.%s/proxy/cloud-controller-ng.service.cf.internal:%d/v2/info?protocol=https", appName, testConfig.AppsDomain, internalCCPort)

		By("checking that our app can't initially reach cloud controller over internal address")
		resp, err := http.Get(proxyRequestURL)
		Expect(err).NotTo(HaveOccurred())

		respBytes, err := io.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		resp.Body.Close()
		Expect(respBytes).To(MatchRegexp("refused"))

		By("creating and binding a security group that allows access to bosh vms for the cc port")
		createASG(asgName, fmt.Sprintf(`[{"destination":"10.0.0.0/0","protocol":"tcp","ports": "%d"}]`, internalCCPort))
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Timeout_Push, func() {
			Expect(cfCLI.BindSecurityGroup(asgName, orgName, spaceName)).To(Succeed())
		})

		if !testConfig.DynamicASGsEnabled {
			By("if dynamic asgs are not enabled, validating an app restart is required")
			Consistently(func() string {
				resp, err = http.Get(proxyRequestURL)
				Expect(err).NotTo(HaveOccurred())

				respBytes, err = io.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())
				resp.Body.Close()
				return string(respBytes)
			}).Should(MatchRegexp("refused"))

			Expect(cf.Cf("restart", appName).Wait(Timeout_Push)).To(gexec.Exit(0))
		}

		By("checking that our app can now reach cloud controller over internal address")
		Eventually(func() string {
			resp, err = http.Get(proxyRequestURL)
			Expect(err).NotTo(HaveOccurred())

			respBytes, err = io.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			resp.Body.Close()
			return string(respBytes)
		}).WithTimeout(180 * time.Second).Should(MatchRegexp("api_version"))

		By("unbinding the security group")
		Expect(cfCLI.UnbindSecurityGroup(asgName, orgName, spaceName)).To(Succeed())

		if !testConfig.DynamicASGsEnabled {
			By("if dynamic asgs are not enabled, validating an app restart is required")
			time.Sleep(10 * time.Second)
			resp, err = http.Get(proxyRequestURL)
			Expect(err).NotTo(HaveOccurred())

			respBytes, err = io.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			resp.Body.Close()
			response := string(respBytes)
			Expect(response).To(MatchRegexp("api_version"))

			Expect(cf.Cf("restart", appName).Wait(Timeout_Push)).To(gexec.Exit(0))
		}

		By("checking that our app can no longer reach cloud controller over internal address")
		Eventually(func() string {
			resp, err = http.Get(proxyRequestURL)
			Expect(err).NotTo(HaveOccurred())

			respBytes, err = io.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			resp.Body.Close()
			return string(respBytes)
		}).WithTimeout(180 * time.Second).Should(MatchRegexp("refused"))
	})
})
