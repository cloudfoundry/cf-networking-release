package integration_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"policy-server/config"
	"policy-server/integration/helpers"

	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"

	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/types"

	"testing"
)

var (
	policyServerPath         string
	policyServerInternalPath string
)

type policyServerPaths struct {
	Internal string
	External string
}

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
	var err error
	paths := policyServerPaths{}
	fmt.Fprintf(GinkgoWriter, "building policy-server binary...")
	paths.External, err = gexec.Build("policy-server/cmd/policy-server", "-race")
	fmt.Fprintf(GinkgoWriter, "done")
	Expect(err).NotTo(HaveOccurred())

	fmt.Fprintf(GinkgoWriter, "building policy-server-internal binary...")
	paths.Internal, err = gexec.Build("policy-server/cmd/policy-server-internal", "-race")
	fmt.Fprintf(GinkgoWriter, "done")
	Expect(err).NotTo(HaveOccurred())

	data, err := json.Marshal(paths)
	Expect(err).NotTo(HaveOccurred())
	return data
}, func(data []byte) {
	var paths policyServerPaths
	err := json.Unmarshal(data, &paths)
	Expect(err).NotTo(HaveOccurred())

	policyServerPath = paths.External
	policyServerInternalPath = paths.Internal

	rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})

func configurePolicyServers(template config.Config, instances int) []config.Config {
	var configs []config.Config
	for i := 0; i < instances; i++ {
		conf := template
		conf.ListenPort = ports.PickAPort()
		conf.DebugServerPort = ports.PickAPort()
		configs = append(configs, conf)
	}
	return configs
}

func configureInternalPolicyServers(template config.InternalConfig, instances int) []config.InternalConfig {
	var configs []config.InternalConfig
	for i := 0; i < instances; i++ {
		conf := template
		conf.InternalListenPort = ports.PickAPort()
		conf.DebugServerPort = ports.PickAPort()
		conf.HealthCheckPort = ports.PickAPort()
		configs = append(configs, conf)
	}
	return configs
}

func startPolicyServers(configs []config.Config) []*gexec.Session {
	return startPolicyAndInternalServers(configs, nil)
}

func startPolicyAndInternalServers(configs []config.Config, internalConfigs []config.InternalConfig) []*gexec.Session {
	testhelpers.CreateDatabase(configs[0].Database)
	var sessions []*gexec.Session
	for _, conf := range configs {
		sessions = append(sessions, helpers.StartPolicyServer(policyServerPath, conf))
	}

	for _, conf := range internalConfigs {
		sessions = append(sessions, helpers.StartInternalPolicyServer(policyServerInternalPath, conf))
	}
	return sessions
}

func stopPolicyServers(sessions []*gexec.Session, configs []config.Config, internalConfigs []config.InternalConfig) {
	for _, session := range sessions {
		session.Interrupt()
		Eventually(session, helpers.DEFAULT_TIMEOUT).Should(gexec.Exit())
	}
	testhelpers.RemoveDatabase(configs[0].Database)
}

func policyServerUrl(route string, confs []config.Config) string {
	conf := confs[rand.Intn(len(confs))]
	return fmt.Sprintf("http://%s:%d/networking/v1/%s", conf.ListenHost, conf.ListenPort, route)
}
