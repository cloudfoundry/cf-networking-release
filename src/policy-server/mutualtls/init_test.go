package mutualtls_test

import (
	"fmt"
	"io/ioutil"
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

	serverCACertPath, err = writeCACert(certstrapBin, certDir, "server-ca")
	Expect(err).NotTo(HaveOccurred())

	serverCertPath, serverKeyPath, err = writeAndSignServerCert(certstrapBin, certDir, "server", "server-ca")
	Expect(err).NotTo(HaveOccurred())

	clientCACertPath, err = writeCACert(certstrapBin, certDir, "client-ca")
	Expect(err).NotTo(HaveOccurred())

	clientCertPath, clientKeyPath, err = writeAndSignServerCert(certstrapBin, certDir, "client", "client-ca")
	Expect(err).NotTo(HaveOccurred())

	wrongClientCACertPath, err = writeCACert(certstrapBin, certDir, "wrong-client-ca")
	Expect(err).NotTo(HaveOccurred())

	wrongClientCertPath, wrongClientKeyPath, err = writeAndSignServerCert(certstrapBin, certDir, "wrong-client", "wrong-client-ca")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	os.Remove(certDir)
})

func TestTls(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mutual TLS Suite")

}

func writeCACert(bin, path, caName string) (string, error) {
	err := exec.Command(bin,
		"--depot-path", path,
		"init",
		"--passphrase", "",
		"--common-name", caName).Run()

	return fmt.Sprintf("%s/%s.crt", path, caName), err
}

func writeAndSignServerCert(bin, path, commonName, caName string) (string, string, error) {
	err := exec.Command(bin,
		"--depot-path", path,
		"request-cert",
		"--passphrase", "",
		"--common-name", commonName,
		"--ip", "127.0.0.1",
		"--domain", commonName).Run()
	if err != nil {
		return "", "", err
	}

	err = exec.Command(bin,
		"--depot-path", path,
		"sign", commonName,
		"--CA", caName).Run()

	return fmt.Sprintf("%s/%s.crt", path, commonName), fmt.Sprintf("%s/%s.key", path, commonName), nil
}

func writeAndSignClientCert(bin, path, commonName, caName string) (string, string, error) {
	err := exec.Command(bin,
		"--depot-path", path,
		"request-cert",
		"--passphrase", "",
		"--common-name", commonName).Run()
	if err != nil {
		return "", "", err
	}

	err = exec.Command(bin,
		"--depot-path", path,
		"sign", commonName,
		"--CA", caName).Run()

	return fmt.Sprintf("%s/%s.crt", path, commonName), fmt.Sprintf("%s/%s.key", path, commonName), nil
}
