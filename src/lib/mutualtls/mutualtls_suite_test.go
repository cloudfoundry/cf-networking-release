package mutualtls_test

import (
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
	certstrapBin          string
	certDir               string
	serverCACertPath      string
	clientCACertPath      string
	serverCertPath        string
	serverKeyPath         string
	clientCertPath        string
	clientKeyPath         string
	wrongClientCACertPath string
	wrongClientCertPath   string
	wrongClientKeyPath    string
)

var _ = BeforeSuite(func() {
	rand.Seed(config.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))

	var err error
	certDir, err = ioutil.TempDir("", "netman-certs")
	Expect(err).NotTo(HaveOccurred())

	certstrapBin = fmt.Sprintf("/%s/certstrap", certDir)
	cmd := exec.Command("go", "build", "-o", certstrapBin, "github.com/square/certstrap")
	Expect(cmd.Run()).NotTo(HaveOccurred())

	certWriter := &testsupport.CertWriter{
		BinPath:  certstrapBin,
		CertPath: certDir,
	}

	serverCACertPath, err = certWriter.WriteCA("server-ca")
	Expect(err).NotTo(HaveOccurred())
	serverCertPath, serverKeyPath, err = certWriter.WriteAndSignForServer("server", "server-ca")
	Expect(err).NotTo(HaveOccurred())

	clientCACertPath, err = certWriter.WriteCA("client-ca")
	Expect(err).NotTo(HaveOccurred())
	clientCertPath, clientKeyPath, err = certWriter.WriteAndSignForClient("client", "client-ca")
	Expect(err).NotTo(HaveOccurred())

	wrongClientCACertPath, err = certWriter.WriteCA("wrong-client-ca")
	Expect(err).NotTo(HaveOccurred())
	wrongClientCertPath, wrongClientKeyPath, err = certWriter.WriteAndSignForClient("wrong-client", "wrong-client-ca")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	os.Remove(certDir)
})

func TestTls(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mutual TLS Suite")
}
