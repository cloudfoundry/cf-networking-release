package integration_test

import (
	"encoding/json"
	"fmt"
	"math/rand"

	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
	"os"
	"path/filepath"
)

var (
	paths testPaths
)

type testPaths struct {
	TeardownBin string
}

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Teardown Integration Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	var err error

	fmt.Fprintf(GinkgoWriter, "building binary...")
	paths.TeardownBin, err = gexec.Build("cni-teardown")
	fmt.Fprintf(GinkgoWriter, "done")
	Expect(err).NotTo(HaveOccurred())

	err = os.Chmod(paths.TeardownBin, 0777)
	Expect(err).NotTo(HaveOccurred())

	dir := filepath.Dir(paths.TeardownBin)
	for ; dir != filepath.Dir(dir); {
		Expect(os.Chmod(dir, 0777)).To(Succeed())
		dir = filepath.Dir(dir)
	}

	data, err := json.Marshal(paths)
	Expect(err).NotTo(HaveOccurred())

	return data
}, func(data []byte) {
	Expect(json.Unmarshal(data, &paths)).To(Succeed())

	rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})
