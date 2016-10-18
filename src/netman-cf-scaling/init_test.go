package scaling_test

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"os/exec"

	pusherConfig "cf-pusher/config"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var (
	config     helpers.Config
	testConfig pusherConfig.Config
)

func TestScaling(t *testing.T) {
	rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))

	RegisterFailHandler(Fail)
	RunSpecs(t, "Scaling Suite")
}

var _ = BeforeSuite(func() {
	config = helpers.LoadConfig()

	configPath := helpers.ConfigPath()
	configBytes, err := ioutil.ReadFile(configPath)
	Expect(err).NotTo(HaveOccurred())

	err = json.Unmarshal(configBytes, &testConfig)
	Expect(err).NotTo(HaveOccurred())

	//TODO see if this property is necessary.
	testConfig.ProxyInstances = 1

	if testConfig.Applications <= 0 {
		Fail("Applications count needs to be greater than 0")
	}

	if testConfig.AppInstances <= 0 {
		Fail("AppInstances count needs to be greater than 0")
	}

	Expect(cf.Cf("api", "--skip-ssl-validation", config.ApiEndpoint).Wait(Timeout_Short)).To(gexec.Exit(0))
	AuthAsAdmin()
	Expect(cf.Cf("target", "-o", testConfig.Prefix+"org", "-s", testConfig.Prefix+"space").Wait(Timeout_Push)).To(gexec.Exit(0))
})

func AuthAsAdmin() {
	Auth(config.AdminUser, config.AdminPassword)
}

func Auth(username, password string) {
	By("authenticating as " + username)
	cmd := exec.Command("cf", "auth", username, password)
	sess, err := gexec.Start(cmd, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess.Wait(Timeout_Short)).Should(gexec.Exit(0))
}
