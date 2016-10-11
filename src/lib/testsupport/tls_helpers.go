package testsupport

import (
	"fmt"
	"os/exec"
)

func WriteCACert(bin, path, caName string) (string, error) {
	err := exec.Command(bin,
		"--depot-path", path,
		"init",
		"--passphrase", "",
		"--common-name", caName).Run()

	return fmt.Sprintf("%s/%s.crt", path, caName), err
}

func WriteAndSignServerCert(bin, path, commonName, caName string) (string, string, error) {
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

func WriteAndSignClientCert(bin, path, commonName, caName string) (string, string, error) {
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
