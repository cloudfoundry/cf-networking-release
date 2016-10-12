package testsupport

import (
	"fmt"
	"os/exec"
)

type CertWriter struct {
	BinPath  string
	CertPath string
}

func (c *CertWriter) WriteCA(caName string) (string, error) {
	err := exec.Command(c.BinPath,
		"--depot-path", c.CertPath,
		"init",
		"--passphrase", "",
		"--common-name", caName).Run()

	return fmt.Sprintf("%s/%s.crt", c.CertPath, caName), err
}

func (c *CertWriter) WriteAndSignForServer(commonName, caName string) (string, string, error) {
	err := exec.Command(c.BinPath,
		"--depot-path", c.CertPath,
		"request-cert",
		"--passphrase", "",
		"--common-name", commonName,
		"--ip", "127.0.0.1",
		"--domain", commonName).Run()
	if err != nil {
		return "", "", err
	}

	err = exec.Command(c.BinPath,
		"--depot-path", c.CertPath,
		"sign", commonName,
		"--CA", caName).Run()

	return fmt.Sprintf("%s/%s.crt", c.CertPath, commonName), fmt.Sprintf("%s/%s.key", c.CertPath, commonName), nil
}

func (c *CertWriter) WriteAndSignForClient(commonName, caName string) (string, string, error) {
	err := exec.Command(c.BinPath,
		"--depot-path", c.CertPath,
		"request-cert",
		"--passphrase", "",
		"--common-name", commonName).Run()
	if err != nil {
		return "", "", err
	}

	err = exec.Command(c.BinPath,
		"--depot-path", c.CertPath,
		"sign", commonName,
		"--CA", caName).Run()

	return fmt.Sprintf("%s/%s.crt", c.CertPath, commonName), fmt.Sprintf("%s/%s.key", c.CertPath, commonName), nil
}
