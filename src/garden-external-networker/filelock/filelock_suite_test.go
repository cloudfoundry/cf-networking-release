package filelock_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestFilelock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Filelock Suite")
}

const demoPackagePath = "garden-external-networker/filelock/filelock-demo"

var pathToBinary string

var _ = SynchronizedBeforeSuite(func() []byte {
	var err error
	pathToBinary, err = gexec.Build(demoPackagePath)
	Expect(err).NotTo(HaveOccurred())
	return []byte(pathToBinary)
}, func(crossNodeData []byte) {
	pathToBinary = string(crossNodeData)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})
