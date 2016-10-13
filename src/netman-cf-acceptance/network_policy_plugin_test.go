package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"sync"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("network policy plugin", func() {
	var (
		appA      string
		appB      string
		orgName   string
		spaceName string
	)

	BeforeEach(func() {
		appA = fmt.Sprintf("appA-%d", rand.Int31())
		appB = fmt.Sprintf("appB-%d", rand.Int31())

		Auth(testConfig.TestUser, testConfig.TestUserPassword)

		orgName = "test-org"
		Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

		spaceName = "test-space"
		Expect(cf.Cf("create-space", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	AfterEach(func() {
		appReport(appA, Timeout_Short)
		appReport(appB, Timeout_Short)

		// clean up everything
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	It("allows the user to create a policy with a cf cli plugin", func() {
		By("pushing appA and appB")
		var setupWG sync.WaitGroup
		setupWG.Add(2)
		go func() {
			defer GinkgoRecover()
			pushApp(appA)
			setupWG.Done()
		}()

		go func() {
			defer GinkgoRecover()
			pushApp(appB)
			setupWG.Done()
		}()
		setupWG.Wait()

		appBIP := getContainerIP(appB, config.AppsDomain)
		By("checking that the default deny is in place")
		assertResponseContains(appBIP, 8080, appA, "request failed")

		By("creating a policy between the apps")
		Expect(cf.Cf("access-allow",
			appA, appB,
			"--port", "8080",
			"--protocol", "tcp").Wait(Timeout_Short)).To(gexec.Exit(0))

		By("checking that the connection succeeds")
		assertResponseContains(appBIP, 8080, appA, "ListenAddresses")

		By("creating a policy between the apps")
		Expect(cf.Cf("access-deny",
			appA, appB,
			"--port", "8080",
			"--protocol", "tcp").Wait(Timeout_Short)).To(gexec.Exit(0))

		By("checking that the connection fails")
		assertResponseContains(appBIP, 8080, appA, "request failed")
	})
})

func getContainerIP(app, domain string) string {
	res, err := http.Get(fmt.Sprintf("http://%s.%s", app, domain))
	Expect(err).NotTo(HaveOccurred())
	defer res.Body.Close()

	bodyBytes, err := ioutil.ReadAll(res.Body)
	Expect(err).NotTo(HaveOccurred())

	var data struct {
		ListenAddresses []string
	}

	err = json.Unmarshal(bodyBytes, &data)
	Expect(err).NotTo(HaveOccurred())

	for _, v := range data.ListenAddresses {
		if strings.HasPrefix(v, "10.255.") {
			return v
		}
	}
	return ""
}
