package mutualtls_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"lib/testsupport"
	"math/rand"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"

	"testing"
)

var (
	certDir string
	paths   testPaths
)

type testPaths struct {
	ServerCACertPath      string
	ClientCACertPath      string
	ServerCertPath        string
	ServerKeyPath         string
	ClientCertPath        string
	ClientKeyPath         string
	WrongClientCACertPath string
	WrongClientCertPath   string
	WrongClientKeyPath    string
}

var _ = SynchronizedBeforeSuite(func() []byte {
	var err error
	certDir, err = ioutil.TempDir("", "netman-certs")
	Expect(err).NotTo(HaveOccurred())

	certstrapBin := fmt.Sprintf("/%s/certstrap", certDir)
	cmd := exec.Command("go", "build", "-o", certstrapBin, "github.com/square/certstrap")
	Expect(cmd.Run()).NotTo(HaveOccurred())

	certWriter := &testsupport.CertWriter{
		BinPath:  certstrapBin,
		CertPath: certDir,
	}

	paths.ServerCACertPath, err = certWriter.WriteCA("server-ca")
	Expect(err).NotTo(HaveOccurred())
	paths.ServerCertPath, paths.ServerKeyPath, err = certWriter.WriteAndSignForServer("server", "server-ca")
	Expect(err).NotTo(HaveOccurred())

	paths.ClientCACertPath, err = certWriter.WriteCA("client-ca")
	Expect(err).NotTo(HaveOccurred())
	paths.ClientCertPath, paths.ClientKeyPath, err = certWriter.WriteAndSignForClient("client", "client-ca")
	Expect(err).NotTo(HaveOccurred())

	paths.WrongClientCACertPath, err = certWriter.WriteCA("wrong-client-ca")
	Expect(err).NotTo(HaveOccurred())
	paths.WrongClientCertPath, paths.WrongClientKeyPath, err = certWriter.WriteAndSignForClient("wrong-client", "wrong-client-ca")
	Expect(err).NotTo(HaveOccurred())

	data, err := json.Marshal(paths)
	Expect(err).NotTo(HaveOccurred())

	return data
}, func(data []byte) {
	Expect(json.Unmarshal(data, &paths)).To(Succeed())

	rand.Seed(config.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	os.Remove(certDir)
})

func TestTls(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mutual TLS Suite")
}
