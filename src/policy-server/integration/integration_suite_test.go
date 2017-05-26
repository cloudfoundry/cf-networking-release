package integration_test

import (
	"fmt"
	"math/rand"
	"policy-server/config"
	"policy-server/integration/helpers"

	"code.cloudfoundry.org/cf-networking-helpers/metrics"

	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/types"

	"testing"
)

var policyServerPath string

var HaveName = func(name string) types.GomegaMatcher {
	return WithTransform(func(ev metrics.Event) string {
		return ev.Name
	}, Equal(name))
}

var mockCCServer = helpers.MockCCServer
var mockUAAServer = helpers.MockUAAServer

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	fmt.Fprintf(GinkgoWriter, "building binary...")
	policyServerPath, err := gexec.Build("policy-server/cmd/policy-server", "-race")
	fmt.Fprintf(GinkgoWriter, "done")
	Expect(err).NotTo(HaveOccurred())

	return []byte(policyServerPath)
}, func(data []byte) {
	policyServerPath = string(data)
	rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})

func configurePolicyServers(template config.Config, instances int) []config.Config {
	var configs []config.Config
	for i := 0; i < instances; i++ {
		conf := template
		conf.ListenPort += i * 100
		conf.InternalListenPort += i * 100
		conf.DebugServerPort += i * 100
		configs = append(configs, conf)
	}
	return configs
}

func startPolicyServers(configs []config.Config) []*gexec.Session {
	var sessions []*gexec.Session
	for _, conf := range configs {
		sessions = append(sessions, helpers.StartPolicyServer(policyServerPath, conf))
	}
	return sessions
}

func stopPolicyServers(sessions []*gexec.Session) {
	for _, session := range sessions {
		session.Interrupt()
		Eventually(session, helpers.DEFAULT_TIMEOUT).Should(gexec.Exit())
	}
}

func policyServerUrl(route string, confs []config.Config) string {
	conf := confs[rand.Intn(len(confs))]
	return fmt.Sprintf("http://%s:%d/networking/v0/%s", conf.ListenHost, conf.ListenPort, route)
}
