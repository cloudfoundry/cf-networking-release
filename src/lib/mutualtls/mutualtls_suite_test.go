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

	paths := testPaths{
		serverCACertPath,
		clientCACertPath,
		serverCertPath,
		serverKeyPath,
		clientCertPath,
		clientKeyPath,
		wrongClientCACertPath,
		wrongClientCertPath,
		wrongClientKeyPath,
	}
	data, err := json.Marshal(paths)
	Expect(err).NotTo(HaveOccurred())

	return data
}, func(data []byte) {

	var paths testPaths
	Expect(json.Unmarshal(data, &paths)).To(Succeed())

	serverCACertPath = paths.ServerCACertPath
	clientCACertPath = paths.ClientCACertPath
	serverCertPath = paths.ServerCertPath
	serverKeyPath = paths.ServerKeyPath
	clientCertPath = paths.ClientCertPath
	clientKeyPath = paths.ClientKeyPath
	wrongClientCACertPath = paths.WrongClientCACertPath
	wrongClientCertPath = paths.WrongClientCertPath
	wrongClientKeyPath = paths.WrongClientKeyPath

	rand.Seed(config.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	os.Remove(certDir)
})

func TestTls(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mutual TLS Suite")
}
