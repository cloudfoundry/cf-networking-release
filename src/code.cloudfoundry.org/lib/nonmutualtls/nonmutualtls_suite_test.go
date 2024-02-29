package nonmutualtls_test

import (
	"encoding/json"
	"os"
	"testing"

	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	certDir string
	paths   testPaths
)

type testPaths struct {
	EmptyFilePath         string
	ServerCACertPath1     string
	ServerCACertPath2     string
	ServerCertPath        string
	ServerKeyPath         string
	WrongServerCACertPath string
}

var _ = SynchronizedBeforeSuite(func() []byte {
	var err error
	certDir, err = os.MkdirTemp("", "netman-certs")
	Expect(err).NotTo(HaveOccurred())

	file, err := os.CreateTemp("", "empty")
	Expect(err).NotTo(HaveOccurred())
	paths.EmptyFilePath = file.Name()

	err = os.WriteFile(paths.EmptyFilePath, []byte("  \n\r\t"), 0600)
	Expect(err).NotTo(HaveOccurred())

	certWriter, err := testsupport.NewCertWriter(certDir)
	Expect(err).NotTo(HaveOccurred())

	paths.ServerCACertPath1, err = certWriter.WriteCA("server-ca-1")
	Expect(err).NotTo(HaveOccurred())
	paths.ServerCACertPath2, err = certWriter.WriteCA("server-ca-2")
	Expect(err).NotTo(HaveOccurred())
	paths.ServerCertPath, paths.ServerKeyPath, err = certWriter.WriteAndSign("server", "server-ca-1")
	Expect(err).NotTo(HaveOccurred())

	paths.WrongServerCACertPath, err = certWriter.WriteCA("wrong-server-ca")
	Expect(err).NotTo(HaveOccurred())

	data, err := json.Marshal(paths)
	Expect(err).NotTo(HaveOccurred())

	return data
}, func(data []byte) {
	Expect(json.Unmarshal(data, &paths)).To(Succeed())
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	os.Remove(certDir)
})

func TestNonmutualtls(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Nonmutualtls Suite")
}
