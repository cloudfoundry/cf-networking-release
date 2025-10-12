package helpers

import (
	"cmp"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"os"
	"strconv"
	"time"

	"code.cloudfoundry.org/lager/v3"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// MYSQL group_concat_max_len system variable
// defines the length of the result returned by GROUP_CONCAT() function
// default value 1024 only allows 282 instance indexes to be concatenated
// this will allow 10_000_000 instance indexes
const (
	MYSQL_GROUP_CONCAT_MAX_LEN = 78888889
	defaultTimeout             = 10 * time.Minute
)

type BBSDBParam struct {
	DriverName                    string
	DatabaseConnectionString      string
	SqlCACertFile                 string
	SqlEnableIdentityVerification bool
	ConnectionTimeout             time.Duration
	ReadTimeout                   time.Duration
	WriteTimeout                  time.Duration
}

func Connect(
	logger lager.Logger, bbsDBParam *BBSDBParam) (*sql.DB, error) {
	connString := addTLSParams(logger, bbsDBParam)
	driverName := bbsDBParam.DriverName

	if driverName == "postgres" {
		driverName = "pgx"
	}

	return sql.Open(driverName, connString)
}

// addTLSParams appends necessary extra parameters to the
// connection string if tls verifications is enabled.  If
// SqlEnableIdentityVerification is true, turn on hostname/identity
// verification, otherwise only ensure that the server certificate is signed by
// one of the CAs in SqlCACertFile.
func addTLSParams(
	logger lager.Logger,
	bbsDBParam *BBSDBParam,
) string {
	dbConnectionString := bbsDBParam.DatabaseConnectionString
	switch bbsDBParam.DriverName {
	case "mysql":
		cfg, err := mysql.ParseDSN(dbConnectionString)
		if err != nil {
			logger.Fatal("invalid-db-connection-string", err, lager.Data{"connection-string": dbConnectionString})
		}

		tlsConfig := generateTLSConfig(logger, bbsDBParam.SqlCACertFile, bbsDBParam.SqlEnableIdentityVerification)
		if tlsConfig != nil {
			err = mysql.RegisterTLSConfig("bbs-tls", tlsConfig)
			if err != nil {
				logger.Fatal("cannot-register-tls-config", err)
			}
			cfg.TLSConfig = "bbs-tls"
		}

		cfg.ReadTimeout = cmp.Or(bbsDBParam.ReadTimeout, defaultTimeout)
		cfg.WriteTimeout = cmp.Or(bbsDBParam.WriteTimeout, defaultTimeout)
		cfg.Timeout = cmp.Or(bbsDBParam.ConnectionTimeout, defaultTimeout)

		cfg.Params = map[string]string{
			"group_concat_max_len": strconv.Itoa(MYSQL_GROUP_CONCAT_MAX_LEN),
		}
		dbConnectionString = cfg.FormatDSN()
	case "postgres":
		config, err := pgx.ParseConfig(dbConnectionString)
		if err != nil {
			logger.Fatal("invalid-db-connection-string", err, lager.Data{"connection-string": dbConnectionString})
		}

		tlsConfig := generateTLSConfig(logger, bbsDBParam.SqlCACertFile, bbsDBParam.SqlEnableIdentityVerification)
		config.TLSConfig = tlsConfig

		dbConnectionString = config.ConnString()
	default:
		logger.Fatal("invalid-driver-name", nil, lager.Data{"driver-name": bbsDBParam.DriverName})
	}

	return dbConnectionString
}

func generateTLSConfig(logger lager.Logger, sqlCACertPath string, SqlEnableIdentityVerification bool) *tls.Config {
	var tlsConfig *tls.Config

	if sqlCACertPath == "" {
		return tlsConfig
	}

	certBytes, err := os.ReadFile(sqlCACertPath)
	if err != nil {
		logger.Fatal("failed-to-read-sql-ca-file", err)
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(certBytes); !ok {
		logger.Fatal("failed-to-parse-sql-ca", err)
	}

	if SqlEnableIdentityVerification {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: false,
			RootCAs:            caCertPool,
		}
	} else {
		tlsConfig = &tls.Config{
			InsecureSkipVerify:    true,
			RootCAs:               caCertPool,
			VerifyPeerCertificate: generateCustomTLSVerificationFunction(caCertPool),
		}
	}

	return tlsConfig
}

func generateCustomTLSVerificationFunction(caCertPool *x509.CertPool) func([][]byte, [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		opts := x509.VerifyOptions{
			Roots:         caCertPool,
			CurrentTime:   time.Now(),
			DNSName:       "",
			Intermediates: x509.NewCertPool(),
		}

		certs := make([]*x509.Certificate, len(rawCerts))
		for i, rawCert := range rawCerts {
			cert, err := x509.ParseCertificate(rawCert)
			if err != nil {
				return err
			}
			certs[i] = cert
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
}
