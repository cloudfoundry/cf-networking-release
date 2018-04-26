package testhelpers

import (
	"crypto/x509"
	"crypto/tls"
	"net/http"
)

func NewClient(caCertPool *x509.CertPool, cert tls.Certificate) *http.Client {
	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		ClientCAs:    caCertPool,
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
	}

	tlsConfig.BuildNameToCertificate()

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &http.Client{Transport: tr}
}


