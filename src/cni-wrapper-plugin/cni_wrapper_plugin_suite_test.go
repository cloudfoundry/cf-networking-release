package main_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestNoop(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CNI wrapper plugin Suite")
}

const packagePath = "cni-wrapper-plugin"
const noopPath = "github.com/containernetworking/cni/plugins/test/noop"

var paths testPaths

type testPaths struct {
	PathToPlugin string
	CNIPath      string
}

var _ = SynchronizedBeforeSuite(func() []byte {

	noopBin, err := gexec.Build(noopPath)
	Expect(err).NotTo(HaveOccurred())
	noopDir, _ := filepath.Split(noopBin)

	pathToPlugin, err := gexec.Build(packagePath)
	Expect(err).NotTo(HaveOccurred())
	wrapperDir, _ := filepath.Split(pathToPlugin)

	paths := testPaths{
		PathToPlugin: pathToPlugin,
		CNIPath:      fmt.Sprintf("%s:%s", wrapperDir, noopDir),
	}

	data, err := json.Marshal(paths)
	Expect(err).NotTo(HaveOccurred())
	return data
}, func(data []byte) {
	Expect(json.Unmarshal(data, &paths)).To(Succeed())
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})
