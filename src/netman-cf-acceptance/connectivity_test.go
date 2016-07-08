package acceptance_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const Timeout_Push = 5 * time.Minute
const Timeout_Short = 10 * time.Second

func getSubnet(ip string) string {
	return strings.Split(ip, ".")[2]
}

func isSameCell(sourceIP, destIP string) bool {
	return getSubnet(sourceIP) == getSubnet(destIP)
}

var _ = Describe("connectivity tests", func() {
	var (
		appA      string
		appB      string
		appAGuid  string
		appBGuid  string
		orgName   string
		spaceName string
		port      int
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

		pushApp(appA)

		pushApp(appB)
		scaleApp(appB, 4 /* instances */)

		appAGuid = getAppGuid(appA)
		appBGuid = getAppGuid(appB)

		port = 8080
	})

	AfterEach(func() {
		// clean up everything
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	Describe("networking policy", func() {
		It("allows the user to configure connections", func() {
			AssertConnectionFails(appA, appB, port)

			By("creating a new policy")
			policyJSON := fmt.Sprintf(`{"policies":[{"source":{"id":"%s"},"destination":{"id":"%s","protocol":"tcp","port":%d}}]}`,
				appAGuid,
				appBGuid,
				port,
			)
			curlSession := cf.Cf("curl", "-X", "POST", "/networking/v0/external/policies", "-d", "'"+policyJSON+"'").Wait(Timeout_Push)
			Expect(curlSession.Wait(Timeout_Push)).To(gexec.Exit(0))
			postPolicyOut := string(curlSession.Out.Contents())
			Expect(postPolicyOut).To(MatchJSON(`{}`))

			AssertConnectionSucceeds(appA, appB, port)

			scaleApp(appA, 4 /* instances */)
			AssertConnectionSucceeds(appA, appB, port)

			scaleApp(appB, 6 /* instances */)
			AssertConnectionSucceeds(appA, appB, port)

			By("deleting the policy")
			curlSession = cf.Cf("curl", "-X", "DELETE", "/networking/v0/external/policies", "-d", "'"+policyJSON+"'").Wait(Timeout_Push)
			Expect(curlSession.Wait(Timeout_Push)).To(gexec.Exit(0))
			deletePolicyOut := string(curlSession.Out.Contents())
			Expect(deletePolicyOut).To(MatchJSON(`{}`))

			time.Sleep(5 * time.Second)
			AssertConnectionFails(appA, appB, port)
		})
	})
})

func assertConnection(sourceAppName string, sourceAppInstance int, destIP string, destPort int, shouldSucceed bool) {
	if shouldSucceed {
		Eventually(func() string {
			return curlFromApp(sourceAppName, sourceAppInstance, fmt.Sprintf("%s:%d/", destIP, destPort), shouldSucceed)
		}, 6*Timeout_Short).Should(ContainSubstring(destIP))
	} else {
		Consistently(func() string {
			return curlFromApp(sourceAppName, sourceAppInstance, fmt.Sprintf("%s:%d/", destIP, destPort), shouldSucceed)
		}, 6*Timeout_Short).ShouldNot(ContainSubstring(destIP))
	}
}

func getInstanceCount(appName string) int {
	appGuid := getAppGuid(appName)
	curlSession := cf.Cf("curl", "-X", "GET", fmt.Sprintf("/v2/apps/%s", appGuid)).Wait(Timeout_Push)
	Expect(curlSession.Wait(Timeout_Push)).To(gexec.Exit(0))
	var infoStruct struct {
		Entity struct {
			Instances int `json:"instances"`
		} `json:"entity"`
	}
	Expect(json.Unmarshal(curlSession.Out.Contents(), &infoStruct)).To(Succeed())
	count := infoStruct.Entity.Instances
	Expect(count).To(BeNumerically(">", 0))
	return count
}

func AssertConnectionSucceeds(sourceApp, destApp string, destPort int) {
	By(fmt.Sprintf("checking that %s can reach %s at port %d", sourceApp, destApp, destPort))
	assertConnectionStatus(sourceApp, destApp, destPort, true)
}

func AssertConnectionFails(sourceApp, destApp string, destPort int) {
	By(fmt.Sprintf("checking that %s can NOT reach %s at port %d", sourceApp, destApp, destPort))
	assertConnectionStatus(sourceApp, destApp, destPort, false)
}

func assertConnectionStatus(sourceApp, destApp string, destPort int, shouldSucceed bool) {
	sourceAppInstances := getInstanceCount(sourceApp)
	destAppInstances := getInstanceCount(destApp)

	sameCellChan := make(chan bool)

	for i := 0; i < sourceAppInstances; i++ {
		for j := 0; j < destAppInstances; j++ {
			go func(sourceAppInstance, destAppInstance int) {
				defer GinkgoRecover()
				sourceIP := getInstanceIP(sourceApp, sourceAppInstance)
				destIP := getInstanceIP(destApp, destAppInstance)

				sameCell := isSameCell(sourceIP, destIP)

				assertConnection(sourceApp, sourceAppInstance, destIP, destPort, shouldSucceed)

				sameCellChan <- sameCell
			}(i, j)
		}
	}

	var coveredSameCell, coveredDifferentCells bool
	for i := 0; i < sourceAppInstances*destAppInstances; i++ {
		sameCell := <-sameCellChan
		if sameCell {
			coveredSameCell = true
		} else {
			coveredDifferentCells = true
		}
	}

	Expect(coveredSameCell).To(BeTrue())
	Expect(coveredDifferentCells).To(BeTrue())
}
