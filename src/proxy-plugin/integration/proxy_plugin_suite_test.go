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
	RunSpecs(t, "Integration: Proxy Plugin Suite")
}

const packagePath = "proxy-plugin/cmd"

var paths testPaths

type testPaths struct {
	PathToPlugin string
	CNIPath      string
}

var _ = SynchronizedBeforeSuite(func() []byte {
	pathToPlugin, err := gexec.Build(packagePath)
	Expect(err).NotTo(HaveOccurred())
	binDir, _ := filepath.Split(pathToPlugin)

	paths := testPaths{
		PathToPlugin: pathToPlugin,
		CNIPath:      fmt.Sprintf("%s", binDir),
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