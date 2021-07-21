package main_test

import (
	"crypto/tls"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
	"time"

	"github.com/nats-io/gnatsd/server"
	gnatsd "github.com/nats-io/gnatsd/test"
	"github.com/onsi/gomega/gexec"
)

func TestServiceDiscoveryController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ServiceDiscoveryController Suite")
}

var pathToServer string

var _ = SynchronizedBeforeSuite(func() []byte {
	path, err := gexec.Build("service-discovery-controller")
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

func RunNatsServerOnPort(port int, tlsConfig *tls.Config) *server.Server {
	opts := gnatsd.DefaultTestOptions
	opts.Port = port
	if tlsConfig != nil {
		opts.TLS = true
		opts.TLSConfig = tlsConfig
	}
	return gnatsd.RunServer(&opts)
}
