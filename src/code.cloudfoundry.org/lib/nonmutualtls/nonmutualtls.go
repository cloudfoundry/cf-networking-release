package nonmutualtls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"
)

func NewServerTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	keyPair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("unable to load cert or key: %s", err)
	}
	c := &tls.Config{
		Certificates: []tls.Certificate{keyPair},
		MinVersion:   tls.VersionTLS12,
	}
	c.PreferServerCipherSuites = true
	c.CipherSuites = []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}
	return c, nil
}

func NewClientTLSConfig(caCertFiles ...string) (*tls.Config, error) {
	c := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	caCertPool := x509.NewCertPool()
	for _, caCertFile := range caCertFiles {
		var err error
		certBytes, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, fmt.Errorf("failed read ca cert file: %s", err.Error())
		}

		if isEmptyBytes(certBytes) {
			continue
		}

		if ok := caCertPool.AppendCertsFromPEM(certBytes); !ok {
			return nil, errors.New("Unable to load caCert")
		}

		c.RootCAs = caCertPool
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

func isEmptyBytes(bytes []byte) bool {
	trimmedStr := strings.Trim(string(bytes), "\t\n\r ")
	return trimmedStr == ""
}
