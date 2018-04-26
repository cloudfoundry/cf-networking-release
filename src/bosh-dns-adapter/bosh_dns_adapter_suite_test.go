package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gexec"
	"testing"
	"time"
)

func TestBoshDnsAdapter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BoshDnsAdapter Suite")
}

var pathToServer string

var _ = SynchronizedBeforeSuite(func() []byte {
	path, err := gexec.Build("bosh-dns-adapter")
	Expect(err).NotTo(HaveOccurred())
	SetDefaultEventuallyTimeout(2 * time.Second)
	return []byte(path)
}, func(data []byte) {
	pathToServer = string(data)
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})
