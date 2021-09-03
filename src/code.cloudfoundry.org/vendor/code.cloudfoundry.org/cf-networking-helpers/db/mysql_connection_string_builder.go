package db

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/go-sql-driver/mysql"
)

//go:generate counterfeiter -o ../fakes/mysql_adapter.go --fake-name MySQLAdapter . mySQLAdapter
type mySQLAdapter interface {
	ParseDSN(dsn string) (cfg *mysql.Config, err error)
	RegisterTLSConfig(key string, config *tls.Config) error
}

type MySQLConnectionStringBuilder struct {
	MySQLAdapter mySQLAdapter
}

func (m *MySQLConnectionStringBuilder) Build(config Config) (string, error) {
	connString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", config.User, config.Password, config.Host, config.Port, config.DatabaseName)

	dbConfig, err := m.MySQLAdapter.ParseDSN(connString)
	if err != nil {
		return "", fmt.Errorf("parsing db connection string: %s", err)
	}

	timeoutDuration := time.Duration(config.Timeout) * time.Second
	dbConfig.Timeout = timeoutDuration
	dbConfig.ReadTimeout = timeoutDuration
	dbConfig.WriteTimeout = timeoutDuration

	if config.RequireSSL {
		dbConfig.TLSConfig = fmt.Sprintf("%s-tls", config.DatabaseName)

		certBytes, err := ioutil.ReadFile(config.CACert)
		if err != nil {
			return "", fmt.Errorf("reading db ca cert file: %s", err)
		}

		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(certBytes); !ok {
			return "", fmt.Errorf("appending cert to pool from pem - invalid cert bytes")
		}

		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			RootCAs:            caCertPool,
		}

		if config.SkipHostnameValidation {
			tlsConfig.InsecureSkipVerify = true

			tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				return VerifyCertificatesIgnoreHostname(rawCerts, caCertPool)
			}
		}

		err = m.MySQLAdapter.RegisterTLSConfig(dbConfig.TLSConfig, tlsConfig)
		if err != nil {
			return "", fmt.Errorf("registering mysql tls config: %s", err)
		}
	}

	return dbConfig.FormatDSN(), nil
}

func VerifyCertificatesIgnoreHostname(rawCerts [][]byte, caCertPool *x509.CertPool) error {
	certs := make([]*x509.Certificate, len(rawCerts))
	for i, asn1Data := range rawCerts {
		cert, err := x509.ParseCertificate(asn1Data)
		if err != nil {
			return fmt.Errorf("tls: failed to parse certificate from server: %s", err)
		}
		certs[i] = cert
	}

	opts := x509.VerifyOptions{
		Roots:         caCertPool,
		CurrentTime:   time.Now(),
		Intermediates: x509.NewCertPool(),
	}

	for i, cert := range certs {
		if i == 0 {
			continue
		}
		opts.Intermediates.AddCert(cert)
	}

	_, err := certs[0].Verify(opts)
	if err != nil {
		return err
	}

	return nil
}
