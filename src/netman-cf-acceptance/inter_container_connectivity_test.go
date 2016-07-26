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

var _ = Describe("connectivity between containers on the overlay network", func() {
	var (
		appA      string
		appB      string
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

		port = 8080
	})

	AfterEach(func() {
		// clean up everything
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	Describe("networking policy", func() {
		It("allows the user to configure connections", func(done Done) {
			AssertConnectionFails(appA, appB, port)

			By("creating a new policy")
			session := cf.Cf("access-allow", appA, appB, "--protocol", "tcp", "--port", fmt.Sprintf("%d", port)).Wait(Timeout_Short)
			Expect(session.Wait(Timeout_Short)).To(gexec.Exit(0))

			AssertConnectionSucceeds(appA, appB, port)

			scaleApp(appA, 4 /* instances */)
			AssertConnectionSucceeds(appA, appB, port)

			scaleApp(appB, 6 /* instances */)
			AssertConnectionSucceeds(appA, appB, port)

			By("deleting the policy")
			session = cf.Cf("access-deny", appA, appB, "--protocol", "tcp", "--port", fmt.Sprintf("%d", port)).Wait(Timeout_Short)
			Expect(session.Wait(Timeout_Short)).To(gexec.Exit(0))

			time.Sleep(5 * time.Second)
			AssertConnectionFails(appA, appB, port)

			close(done)
		}, 600 /* <-- overall spec timeout in seconds */)
	})
})

func assertConnection(sourceAppName string, sourceAppInstance int, destIP string, destPort int, shouldSucceed bool) {
	if shouldSucceed {
		Expect(curlFromApp(sourceAppName, sourceAppInstance, fmt.Sprintf("%s:%d/", destIP, destPort), shouldSucceed)).To(ContainSubstring(destIP))
	} else {
		Expect(curlFromApp(sourceAppName, sourceAppInstance, fmt.Sprintf("%s:%d/", destIP, destPort), shouldSucceed)).NotTo(ContainSubstring(destIP))
	}
}

func getInstanceCount(appName string) int {
	appGuid := getAppGuid(appName)
	curlSession := cf.Cf("curl", "-X", "GET", fmt.Sprintf("/v2/apps/%s", appGuid)).Wait(Timeout_Short)
	Expect(curlSession.Wait(Timeout_Short)).To(gexec.Exit(0))
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

	sources := []string{}
	for i := 0; i < sourceAppInstances; i++ {
		sources = append(sources, getInstanceIP(sourceApp, i))
	}
	dests := []string{}
	for j := 0; j < destAppInstances; j++ {
		dests = append(dests, getInstanceIP(destApp, j))
	}

	sameCellChan := make(chan bool)

	for sourceAppInstance, sourceIP := range sources {
		for _, destIP := range dests {
			go func(sourceIP, destIP string, sourceAppInstance int) {
				defer GinkgoRecover()

				sameCell := isSameCell(sourceIP, destIP)

				assertConnection(sourceApp, sourceAppInstance, destIP, destPort, shouldSucceed)

				sameCellChan <- sameCell
			}(sourceIP, destIP, sourceAppInstance)
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
