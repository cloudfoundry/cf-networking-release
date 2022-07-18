package main_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

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
	fmt.Fprintf(GinkgoWriter, "building binary...")
	wd, err := os.Getwd()
	Expect(err).To(Succeed())
	appPath, err := gexec.Build("tick")
	Expect(err).NotTo(HaveOccurred())

	modPath := filepath.Join("..", "registry")
	Expect(os.Chdir(modPath)).To(Succeed())
	regPath, err := gexec.Build("registry")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.Chdir(wd)).To(Succeed())

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

	rand.Seed(config.GinkgoConfig.RandomSeed + int64(GinkgoParallelProcess()))
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})

func TestTick(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tick Suite")
}
