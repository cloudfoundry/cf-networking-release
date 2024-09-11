package testhelpers

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
)

func NewClient(caCertPool *x509.CertPool, cert tls.Certificate) *http.Client {
	tr := &http.Transport{
		TLSClientConfig: TLSClientConfig(caCertPool, cert),
	}

	return &http.Client{Transport: tr}
}

func TLSClientConfig(caCertPool *x509.CertPool, cert tls.Certificate) *tls.Config {
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		ClientCAs:    caCertPool,
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
	}
}
