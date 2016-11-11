package acceptance_test

import (
	"cf-pusher/cf_cli_adapter"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("policy cleanup", func() {
	var (
		appA, appB, appC string
		orgName          string
		spaceName        string
		cfCli            *cf_cli_adapter.Adapter
	)

	BeforeEach(func() {
		appA = fmt.Sprintf("appA-%d", rand.Int31())
		appB = fmt.Sprintf("appB-%d", rand.Int31())
		appC = fmt.Sprintf("appC-%d", rand.Int31())

		cfCli = &cf_cli_adapter.Adapter{
			CfCliPath: "cf",
		}
		AuthAsAdmin()

		orgName = "cleanup-org"
		Expect(cfCli.CreateOrg(orgName)).To(Succeed())
		Expect(cfCli.TargetOrg(orgName)).To(Succeed())

		spaceName = "cleanup-space"
		Expect(cfCli.CreateSpace(spaceName)).To(Succeed())
		Expect(cfCli.TargetSpace(spaceName)).To(Succeed())

		pushProxy(appA)
		pushProxy(appB)
		pushProxy(appC)
	})

	AfterEach(func() {
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	Describe("policies/cleanup endpoint", func() {
		It("returns stale policies for deleted apps", func() {
			By("creating policies for all apps")
			Expect(cfCli.AllowAccess(appA, appB, 1234, "tcp")).To(Succeed())
			Expect(cfCli.AllowAccess(appB, appC, 1234, "tcp")).To(Succeed())
			Expect(cfCli.AllowAccess(appC, appA, 1234, "tcp")).To(Succeed())

			appAGuid, err := cfCli.AppGuid(appA)
			Expect(err).NotTo(HaveOccurred())

			appCGuid, err := cfCli.AppGuid(appC)
			Expect(err).NotTo(HaveOccurred())

			By("getting all policies")
			allPolicies, err := cfCli.Curl("GET", "/networking/v0/external/policies", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(allPolicies)).Should(ContainSubstring(appCGuid))
			Expect(string(allPolicies)).Should(ContainSubstring(appAGuid))

			By("deleting appC")
			Expect(cfCli.Delete(appC)).To(Succeed())

			By("checking for stale policies")
			stalePolicies, err := cfCli.Curl("POST", "/networking/v0/external/policies/cleanup", "")
			Expect(err).NotTo(HaveOccurred())
			fmt.Println(string(stalePolicies))

			tmpfile, err := ioutil.TempFile("", "stalepolicies")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tmpfile.Name())

			_, err = tmpfile.Write(stalePolicies)
			Expect(err).NotTo(HaveOccurred())
			Expect(tmpfile.Close()).To(Succeed())

			By("deleting stale policies")
			_, err = cfCli.Curl("DELETE", "/networking/v0/external/policies", tmpfile.Name())
			Expect(err).NotTo(HaveOccurred())

			By("getting all policies")
			allPolicies, err = cfCli.Curl("GET", "/networking/v0/external/policies", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(allPolicies)).ShouldNot(ContainSubstring(appCGuid))
			Expect(string(allPolicies)).Should(ContainSubstring(appAGuid))
		})
	})
})
