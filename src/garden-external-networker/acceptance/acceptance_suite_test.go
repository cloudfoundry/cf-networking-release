package acceptance_test

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/config"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Acceptance Suite")
}

var (
	pathToAdapter  string
	cniPluginDir   string
	cniPluginNames []string
)

var _ = BeforeSuite(func() {
	rand.Seed(config.GinkgoConfig.RandomSeed)

	var err error
	pathToAdapter, err = gexec.Build("garden-external-networker", "-race")
	Expect(err).NotTo(HaveOccurred())

	pathToFakeCNIPlugin, err := gexec.Build("garden-external-networker/acceptance/fake-cni-plugin", "-race")
	Expect(err).NotTo(HaveOccurred())

	cniPluginDir, err = ioutil.TempDir("", "cni-plugin-")
	Expect(err).NotTo(HaveOccurred())

	cniPluginNames = []string{"plugin-0", "plugin-1", "plugin-2", "plugin-3"}
	for _, name := range cniPluginNames {
		os.Link(pathToFakeCNIPlugin, filepath.Join(cniPluginDir, name))
	}
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	Expect(os.RemoveAll(cniPluginDir)).To(Succeed())
})
