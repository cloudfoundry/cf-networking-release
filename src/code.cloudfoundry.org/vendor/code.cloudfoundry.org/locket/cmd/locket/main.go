package main

import (
	"context"
	"flag"
	"os"
	"time"

	"code.cloudfoundry.org/bbs/db/sqldb/helpers"
	"code.cloudfoundry.org/bbs/db/sqldb/helpers/monitor"
	"code.cloudfoundry.org/bbs/guidprovider"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/debugserver"
	loggingclient "code.cloudfoundry.org/diego-logging-client"
	"code.cloudfoundry.org/go-loggregator/v8/runtimeemitter"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"
	"code.cloudfoundry.org/locket"
	"code.cloudfoundry.org/locket/cmd/locket/config"
	"code.cloudfoundry.org/locket/db"
	"code.cloudfoundry.org/locket/expiration"
	"code.cloudfoundry.org/locket/grpcserver"
	"code.cloudfoundry.org/locket/handlers"
	"code.cloudfoundry.org/locket/metrics"
	metrics_helpers "code.cloudfoundry.org/locket/metrics/helpers"
	"code.cloudfoundry.org/tlsconfig"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

var configFilePath = flag.String(
	"config",
	"",
	"Path to Locket JSON Configuration file",
)

func main() {
	flag.Parse()

	cfg, err := config.NewLocketConfig(*configFilePath)
	if err != nil {
		panic("invalid-config-file: " + err.Error())
	}

	logger, reconfigurableSink := lagerflags.NewFromConfig("locket", cfg.LagerConfig)

	metronClient, err := initializeMetron(logger, cfg)
	if err != nil {
		logger.Error("failed-to-initialize-metron-client", err)
		os.Exit(1)
	}

	clock := clock.NewClock()

	sqlConn, err := helpers.Connect(
		logger,
		cfg.DatabaseDriver,
		cfg.DatabaseConnectionString,
		cfg.SQLCACertFile,
		cfg.SQLEnableIdentityVerification,
	)

	if err != nil {
		logger.Fatal("failed-to-open-sql", err)
	}
	defer sqlConn.Close()

	sqlConn.SetMaxIdleConns(cfg.MaxOpenDatabaseConnections)
	sqlConn.SetMaxOpenConns(cfg.MaxOpenDatabaseConnections)
	sqlConn.SetConnMaxLifetime(time.Duration(cfg.MaxDatabaseConnectionLifetime))

	err = sqlConn.Ping()
	if err != nil {
		logger.Fatal("sql-failed-to-connect", err)
	}

	dbMonitor := monitor.New()
	monitoredDB := helpers.NewMonitoredDB(sqlConn, dbMonitor)

	sqlDB := db.NewSQLDB(
		monitoredDB,
		cfg.DatabaseDriver,
		guidprovider.DefaultGuidProvider,
	)

	err = sqlDB.CreateLockTable(context.Background(), logger)
	if err != nil {
		logger.Fatal("failed-to-create-lock-table", err)
	}

	tlsConfig, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(cfg.CertFile, cfg.KeyFile),
	).Server(tlsconfig.WithClientAuthenticationFromFile(cfg.CaFile))
	if err != nil {
		logger.Fatal("invalid-tls-config", err)
	}

	lockMetricsNotifier := metrics.NewLockMetricsNotifier(logger, clock, metronClient, time.Duration(cfg.ReportInterval), sqlDB)
	dbMetricsNotifier := metrics.NewDBMetricsNotifier(logger, clock, metronClient, time.Duration(cfg.ReportInterval), sqlDB, dbMonitor)
	requestNotifier := metrics_helpers.NewRequestMetricsNotifier(logger, clock, metronClient, time.Duration(cfg.ReportInterval), []string{"Lock", "Release", "Fetch", "FetchAll"})
	lockPick := expiration.NewLockPick(sqlDB, clock, metronClient)
	burglar := expiration.NewBurglar(logger, sqlDB, lockPick, clock, locket.RetryInterval, metronClient)
	exitCh := make(chan struct{})
	handler := handlers.NewLocketHandler(logger, sqlDB, lockPick, requestNotifier, exitCh)
	server := grpcserver.NewGRPCServer(logger, cfg.ListenAddress, tlsConfig, handler)

	members := grouper.Members{
		{"server", server},
		{"burglar", burglar},
		{"lock-metrics-notifier", lockMetricsNotifier},
		{"db-metrics-notifier", dbMetricsNotifier},
		{"request-metrics-notifier", requestNotifier},
	}

	if cfg.DebugAddress != "" {
		members = append(grouper.Members{
			{"debug-server", debugserver.Runner(cfg.DebugAddress, reconfigurableSink)},
		}, members...)
	}

	group := grouper.NewOrdered(os.Interrupt, members)
	monitor := ifrit.Invoke(sigmon.New(group))

	logger.Info("started")

	go func() {
		<-exitCh
		logger.Info("shutting-down-due-to-unrecoverable-error")
		monitor.Signal(os.Interrupt)
	}()

	err = <-monitor.Wait()
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}
}

func initializeMetron(logger lager.Logger, locketConfig config.LocketConfig) (loggingclient.IngressClient, error) {
	client, err := loggingclient.NewIngressClient(locketConfig.LoggregatorConfig)
	if err != nil {
		return nil, err
	}

	if locketConfig.LoggregatorConfig.UseV2API {
		emitter := runtimeemitter.NewV1(client)
		go emitter.Run()
	}

	return client, nil
}
