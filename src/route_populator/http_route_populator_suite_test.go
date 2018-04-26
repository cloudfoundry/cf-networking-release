package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var (
	httpRoutePopulatorPath string
)

func TestHTTPRoutePopulator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTPRoutePopulator Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	routePopulator, err := gexec.Build("route_populator", "-race")
	Expect(err).ToNot(HaveOccurred())

	return []byte(routePopulator)
}, func(payload []byte) {
	httpRoutePopulatorPath = string(payload)
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})
