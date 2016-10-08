package mutualtls_test

import (
	"log"
	"math/rand"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTls(t *testing.T) {
	rand.Seed(config.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))

	RegisterFailHandler(Fail)
	cmd := exec.Command("go", "build", "-o", "/tmp/certstrap", "github.com/square/certstrap")
	err := cmd.Run()
	if err != nil {
		log.Printf("error building certstrap: %s", err)
	}

	defer os.Remove("/tmp/certstrap")
	defer os.Remove("/tmp/netman-ca.crl")
	defer os.Remove("/tmp/netman-ca.crt")
	defer os.Remove("/tmp/netman-ca.key")
	defer os.Remove("/tmp/server.crt")
	defer os.Remove("/tmp/server.csr")
	defer os.Remove("/tmp/server.key")
	defer os.Remove("/tmp/client.crt")
	defer os.Remove("/tmp/client.csr")
	defer os.Remove("/tmp/client.key")
	defer os.Remove("/tmp/wrong-netman-ca.crl")
	defer os.Remove("/tmp/wrong-netman-ca.crt")
	defer os.Remove("/tmp/wrong-netman-ca.key")
	defer os.Remove("/tmp/wrong-client.crt")
	defer os.Remove("/tmp/wrong-client.csr")
	defer os.Remove("/tmp/wrong-client.key")

	writeValidSpecCerts()
	writeWrongClientCerts()

	RunSpecs(t, "Mutual TLS Suite")
}

func writeValidSpecCerts() {
	cmd := exec.Command("/tmp/certstrap",
		"--depot-path", "/tmp",
		"init",
		"--passphrase", "",
		"--common-name", "netman-ca")
	err := cmd.Run()
	if err != nil {
		log.Printf("error creating CA: %s", err)
	}

	cmd = exec.Command("/tmp/certstrap",
		"--depot-path", "/tmp",
		"request-cert",
		"--passphrase", "",
		"--common-name", "server",
		"--ip", "127.0.0.1",
		"--domain", "server")
	err = cmd.Run()
	if err != nil {
		log.Printf("error creating server certs: %s", err)
	}

	cmd = exec.Command("/tmp/certstrap",
		"--depot-path", "/tmp",
		"sign", "server",
		"--CA", "netman-ca")
	err = cmd.Run()
	if err != nil {
		log.Printf("error signing server cert: %s", err)
	}

	cmd = exec.Command("/tmp/certstrap",
		"--depot-path", "/tmp",
		"request-cert",
		"--passphrase", "",
		"--common-name", "client")
	err = cmd.Run()
	if err != nil {
		log.Printf("error creating client cert: %s", err)
	}

	cmd = exec.Command("/tmp/certstrap",
		"--depot-path", "/tmp",
		"sign", "client",
		"--CA", "netman-ca")
	err = cmd.Run()
	if err != nil {
		log.Printf("error signing client cert: %s", err)
	}
}

func writeWrongClientCerts() {
	cmd := exec.Command("/tmp/certstrap",
		"--depot-path", "/tmp",
		"init",
		"--passphrase", "",
		"--common-name", "wrong-netman-ca")
	err := cmd.Run()
	if err != nil {
		log.Printf("error creating CA: %s", err)
	}

	cmd = exec.Command("/tmp/certstrap",
		"--depot-path", "/tmp",
		"request-cert",
		"--passphrase", "",
		"--common-name", "wrong-client")
	err = cmd.Run()
	if err != nil {
		log.Printf("error creating client cert: %s", err)
	}

	cmd = exec.Command("/tmp/certstrap",
		"--depot-path", "/tmp",
		"sign", "wrong-client",
		"--CA", "wrong-netman-ca")
	err = cmd.Run()
	if err != nil {
		log.Printf("error signing client cert: %s", err)
	}
}
