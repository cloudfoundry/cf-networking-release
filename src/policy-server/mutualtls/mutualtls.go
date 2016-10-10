package mutualtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
)

func BuildConfig(serverCert, serverKey, clientCACert []byte) (*tls.Config, error) {
	keyPair, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, fmt.Errorf("unable to load server cert or key: %s", err)
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(clientCACert)
	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{keyPair},
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                certPool,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
		CipherSuites:             []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
	}
	return tlsConfig, nil
}
