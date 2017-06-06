package main_test

import (
	"encoding/json"
	"math/rand"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

const DEFAULT_TIMEOUT = "5s"

var binaryPath string
var registryBinaryPath string

type testApps struct {
	AppPath string
	RegPath string
}

var _ = SynchronizedBeforeSuite(func() []byte {
	appPath, err := gexec.Build("example-apps/tick")
	Expect(err).NotTo(HaveOccurred())

	regPath, err := gexec.Build("../registry")
	Expect(err).NotTo(HaveOccurred())

	apps := testApps{
		appPath,
		regPath,
	}
	bytes, err := json.Marshal(apps)
	Expect(err).NotTo(HaveOccurred())

	return bytes
}, func(data []byte) {

	var apps testApps
	Expect(json.Unmarshal(data, &apps)).To(Succeed())

	binaryPath = apps.AppPath
	registryBinaryPath = apps.RegPath

	rand.Seed(config.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})

func TestTick(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tick Suite")
}
