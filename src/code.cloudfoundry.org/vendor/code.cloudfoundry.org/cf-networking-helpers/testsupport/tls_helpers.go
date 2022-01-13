package testsupport

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"time"

	"github.com/square/certstrap/depot"
	"github.com/square/certstrap/pkix"
)

type CertWriter struct {
	CertPath  string
	fileDepot *depot.FileDepot
}

const RSABits = 1024

func formatName(name string) string {
	return strings.Replace(name, " ", "_", -1)
}

func NewCertWriter(certPath string) (*CertWriter, error) {
	var (
		d   *depot.FileDepot
		err error
	)

	if d, err = depot.NewFileDepot(certPath); err != nil {
		return nil, fmt.Errorf("initialize depot: %s", err)
	}

	return &CertWriter{
		CertPath:  certPath,
		fileDepot: d,
	}, nil
}

func (c *CertWriter) WriteCA(caName string) (string, error) {
	key, err := pkix.CreateRSAKey(RSABits)
	if err != nil {
		return "", fmt.Errorf("create rsa key: %s", err)
	}

	expiry := time.Now().AddDate(5, 0, 0).UTC()
	crt, err := pkix.CreateCertificateAuthority(key, "", expiry, "", "", "", "", caName)
	if err != nil {
		return "", fmt.Errorf("create certificate authority: %s", err)
	}

	formattedCAName := formatName(caName)
	if err = depot.PutCertificate(c.fileDepot, formattedCAName, crt); err != nil {
		return "", fmt.Errorf("save certificate: %s", err)
	}

	if err = depot.PutPrivateKey(c.fileDepot, formattedCAName, key); err != nil {
		return "", fmt.Errorf("save private key: %s", err)
	}

	return fmt.Sprintf("%s/%s.crt", c.CertPath, formattedCAName), nil
}

func (c *CertWriter) WriteAndSign(commonName, caName string) (string, string, error) {
	key, err := pkix.CreateRSAKey(RSABits)
	if err != nil {
		return "", "", fmt.Errorf("create rsa key: %s", err)
	}

	csr, err := pkix.CreateCertificateSigningRequest(
		key, "", []net.IP{net.ParseIP("127.0.0.1")},
		[]string{commonName}, []*url.URL{}, "", "", "", "", commonName,
	)
	if err != nil {
		return "", "", fmt.Errorf("create certificate request: %s", err)
	}

	formattedCommonName := formatName(commonName)
	if err = depot.PutPrivateKey(c.fileDepot, formattedCommonName, key); err != nil {
		return "", "", fmt.Errorf("save private key: %s", err)
	}

	formattedCAName := formatName(caName)
	crt, err := depot.GetCertificate(c.fileDepot, formattedCAName)
	if err != nil {
		return "", "", fmt.Errorf("get certificate: %s", err)
	}

	caKey, err := depot.GetPrivateKey(c.fileDepot, formattedCAName)
	if err != nil {
		return "", "", fmt.Errorf("get CA key: %s", err)
	}

	expiry := time.Now().AddDate(1, 0, 0).UTC()
	crtOut, err := pkix.CreateCertificateHost(crt, caKey, csr, expiry)
	if err != nil {
		return "", "", fmt.Errorf("create certificate host: %s", err)
	}

	if err = depot.PutCertificate(c.fileDepot, formattedCommonName, crtOut); err != nil {
		return "", "", fmt.Errorf("save certificate error: %s", err)
	}

	return fmt.Sprintf("%s/%s.crt", c.CertPath, formattedCommonName), fmt.Sprintf("%s/%s.key", c.CertPath, formattedCommonName), err
}
