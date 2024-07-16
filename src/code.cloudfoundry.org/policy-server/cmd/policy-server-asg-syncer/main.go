package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/json_client"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/mutualtls"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagerflags"
	"code.cloudfoundry.org/lib/common"
	"code.cloudfoundry.org/lib/nonmutualtls"
	"code.cloudfoundry.org/locket"
	"code.cloudfoundry.org/locket/lock"
	locketmodels "code.cloudfoundry.org/locket/models"
	"code.cloudfoundry.org/policy-server/asg_syncer"
	"code.cloudfoundry.org/policy-server/cc_client"
	"code.cloudfoundry.org/policy-server/config"
	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/uaa_client"
	"github.com/cloudfoundry/dropsonde"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

const (
	jobPrefix = "policy-server-asg-syncer"
)

var (
	logPrefix = "cfnetworking"
)

func main() {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	conf, err := config.NewASGSyncer(*configFilePath)
	if err != nil {
		log.Fatalf("%s.%s: could not read config file: %s", logPrefix, jobPrefix, err)
	}

	if conf.LogPrefix != "" {
		logPrefix = conf.LogPrefix
	}

	loggerConfig := common.GetLagerConfig()
	if conf.LogLevel != "" {
		loggerConfig.LogLevel = conf.LogLevel
	}
	logger, _ := lagerflags.NewFromConfig(fmt.Sprintf("%s.%s", logPrefix, jobPrefix), loggerConfig)

	connectionPool, err := db.NewConnectionPool(
		conf.Database,
		1,
		1,
		0,
		logPrefix,
		jobPrefix,
		logger,
	)
	if err != nil {
		log.Fatal(err.Error())
	}

	securityGroupsStore := &store.SGStore{
		Conn: connectionPool,
	}

	metricsSender := &metrics.MetricsSender{
		Logger: logger.Session("time-metric-emitter"),
	}

	err = dropsonde.Initialize(conf.MetronAddress, jobPrefix)
	if err != nil {
		log.Fatalf("%s.%s: initializing dropsonde: %s", logPrefix, jobPrefix, err)
	}

	wrappedSecurityGroupsStore := &store.SecurityGroupsMetricsWrapper{
		Store:         securityGroupsStore,
		MetricsSender: metricsSender,
	}

	locketClient, err := locket.NewClient(logger, conf.ClientLocketConfig)
	if err != nil {
		log.Fatalf("%s.%s: failed-to-create-locket-client using: %s", logPrefix, jobPrefix, err)
	}

	var tlsConfig *tls.Config
	var mutualTlsConfig *tls.Config
	if conf.SkipSSLValidation {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: conf.SkipSSLValidation,
		}

		mutualTlsConfig = &tls.Config{
			InsecureSkipVerify: conf.SkipSSLValidation,
		}
	} else {
		tlsConfig, err = nonmutualtls.NewClientTLSConfig(conf.UAACA, conf.CCCA)
		if err != nil {
			log.Fatalf("%s.%s error creating tls config: %s", logPrefix, jobPrefix, err) // not tested
		}

		mutualTlsConfig, err = mutualtls.NewClientTLSConfig(conf.CCInternalClientCert, conf.CCInternalClientKey, conf.CCInternalCA)
		if err != nil {
			log.Fatalf("%s.%s error creating mutual tls config: %s", logPrefix, jobPrefix, err) // not tested
		}
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	httpInternalClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: mutualTlsConfig,
		},
	}

	uaaClient := &uaa_client.Client{
		BaseURL:    fmt.Sprintf("%s:%d", conf.UAAURL, conf.UAAPort),
		Name:       conf.UAAClient,
		Secret:     conf.UAAClientSecret,
		HTTPClient: httpClient,
		Logger:     logger,
	}

	ccClient := &cc_client.Client{
		ExternalJSONClient: json_client.New(logger.Session("cc-json-client"), httpClient, conf.CCURL),
		InternalJSONClient: json_client.New(logger.Session("cc-internal-json-client"), httpInternalClient, conf.CCInternalURL),
		Logger:             logger,
	}

	asgSyncer := asg_syncer.NewASGSyncer(logger, wrappedSecurityGroupsStore, uaaClient, ccClient, time.Duration(conf.ASGSyncInterval)*time.Second, metricsSender, time.Second*time.Duration(conf.RetryDeadline))
	lock := initASGLocker(logger, conf.UUID, locket.RetryInterval, locket.DefaultSessionTTLInSeconds, locketClient)

	members := grouper.Members{
		{Name: "asg-lock", Runner: lock},
		{Name: "asg-syncer", Runner: asgSyncer},
	}

	logger.Info("starting-asg-syncer", lager.Data{"interval": conf.ASGSyncInterval})

	group := grouper.NewOrdered(os.Interrupt, members)
	monitor := ifrit.Invoke(sigmon.New(group))

	err = <-monitor.Wait()
	if connectionPool != nil {
		connectionPool.Close()
	}
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}

	logger.Info("exited")
}

func initASGLocker(logger lager.Logger, uuid string, lockTimeout time.Duration, lockTTL int64, locketClient locketmodels.LocketClient) ifrit.Runner {
	lockIdentifier := &locketmodels.Resource{
		Key:      "policy-server-asg-syncer",
		Owner:    uuid,
		TypeCode: locketmodels.LOCK,
		Type:     locketmodels.LockType,
	}
	return lock.NewLockRunner(
		logger,
		locketClient,
		lockIdentifier,
		lockTTL,
		clock.NewClock(),
		lockTimeout,
	)
}
