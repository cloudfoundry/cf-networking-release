package mutualtls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
)

func NewServerTLSConfig(certFile, keyFile, caCertFile string) (*tls.Config, error) {
	c, err := newTLSConfig(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	c.ClientCAs, err = newCACertPool(caCertFile)
	if err != nil {
		return nil, err
	}
	c.ClientAuth = tls.RequireAndVerifyClientCert
	c.PreferServerCipherSuites = true
	c.CipherSuites = []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}
	return c, nil
}

func NewClientTLSConfig(certFile, keyFile, caCertFile string) (*tls.Config, error) {
	c, err := newTLSConfig(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	c.RootCAs, err = newCACertPool(caCertFile)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func newTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	keyPair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("unable to load cert or key: %s", err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{keyPair},
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS12,
	}
	return tlsConfig, nil
}

func newCACertPool(caCertFile string) (*x509.CertPool, error) {
	certBytes, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return nil, fmt.Errorf("failed read ca cert file: %s", err.Error())
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(certBytes); !ok {
		return nil, errors.New("Unable to load caCert")
	}
	return caCertPool, nil
}
