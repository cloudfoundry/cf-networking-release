package mutualtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
)

func BuildConfig(serverCert, serverKey, clientCACert []byte) (*tls.Config, error) {
	keyPair, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, fmt.Errorf("unable to load cert or key: %s", err)
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(clientCACert)
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{keyPair},
		ClientCAs:    certPool,
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS12,
	}
	return tlsConfig, nil
}

func BuildServerConfig(serverCert, serverKey, clientCACert []byte) (*tls.Config, error) {
	c, err := BuildConfig(serverCert, serverKey, clientCACert)
	if err != nil {
		return nil, err
	}
	c.ClientAuth = tls.RequireAndVerifyClientCert
	c.PreferServerCipherSuites = true
	c.CipherSuites = []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}
	return c, nil
}
