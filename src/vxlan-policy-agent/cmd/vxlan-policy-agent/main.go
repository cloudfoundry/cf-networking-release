package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"lib/datastore"
	"lib/filelock"
	"lib/metrics"
	"lib/policy_client"
	"lib/poller"
	"lib/rules"
	"lib/serial"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"vxlan-policy-agent/config"
	"vxlan-policy-agent/converger"
	"vxlan-policy-agent/enforcer"
	"vxlan-policy-agent/handlers"
	"vxlan-policy-agent/planner"

	"code.cloudfoundry.org/debugserver"
	"code.cloudfoundry.org/go-db-helpers/mutualtls"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/dropsonde"
	"github.com/coreos/go-iptables/iptables"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

const (
	dropsondeOrigin = "vxlan-policy-agent"
	emitInterval    = 30 * time.Second
)

func die(logger lager.Logger, action string, err error) {
	logger.Error(action, err)
	os.Exit(1)
}

func main() {
	conf := &config.VxlanPolicyAgent{}

	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	configBytes, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		log.Fatalf("error reading config: %s", err)
	}

	err = json.Unmarshal(configBytes, conf)
	if err != nil {
		log.Fatalf("error unmarshalling config: %s", err)
	}

	logger := lager.NewLogger("container-networking.vxlan-policy-agent")
	reconfigurableSink := initLoggerSink(logger, conf.LogLevel)
	logger.RegisterSink(reconfigurableSink)

	logger.Info("parsed-config", lager.Data{"config": conf})

	pollInterval := time.Duration(conf.PollInterval) * time.Second
	if pollInterval == 0 {
		pollInterval = time.Second
	}

	clientTLSConfig, err := mutualtls.NewClientTLSConfig(conf.ClientCertFile, conf.ClientKeyFile, conf.ServerCACertFile)
	if err != nil {
		die(logger, "mutual tls config", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: clientTLSConfig,
		},
		Timeout: time.Duration(conf.ClientTimeoutSeconds) * time.Second,
	}

	policyClient := policy_client.NewInternal(
		logger.Session("policy-client"),
		httpClient,
		conf.PolicyServerURL,
	)

	store := &datastore.Store{
		Serializer: &serial.Serial{},
		Locker: &filelock.Locker{
			Path: conf.Datastore,
		},
	}

	ipt, err := iptables.New()
	if err != nil {
		die(logger, "iptables-new", err)
	}

	iptLocker := &rules.IPTablesLocker{
		FileLocker: &filelock.Locker{Path: conf.IPTablesLockFile},
		Mutex:      &sync.Mutex{},
	}
	restorer := &rules.Restorer{}
	lockedIPTables := &rules.LockedIPTables{
		IPTables: ipt,
		Locker:   iptLocker,
		Restorer: restorer,
	}

	metricsSender := &metrics.MetricsSender{
		Logger: logger.Session("time-metric-emitter"),
	}

	iptablesLoggingState := &planner.LoggingState{}
	if conf.IPTablesLogging {
		iptablesLoggingState.Enable()
	}

	dynamicPlanner := &planner.VxlanPolicyPlanner{
		Datastore:     store,
		PolicyClient:  policyClient,
		Logger:        logger.Session("rules-updater"),
		VNI:           conf.VNI,
		MetricsSender: metricsSender,
		Chain: enforcer.Chain{
			Table:       "filter",
			ParentChain: "FORWARD",
			Prefix:      "vpa--",
		},
		LoggingState: iptablesLoggingState,
	}

	timestamper := &enforcer.Timestamper{}
	ruleEnforcer := enforcer.NewEnforcer(
		logger.Session("rules-enforcer"),
		timestamper,
		lockedIPTables,
	)

	err = dropsonde.Initialize(conf.MetronAddress, dropsondeOrigin)
	if err != nil {
		log.Fatalf("initializing dropsonde: %s", err)
	}

	uptimeSource := metrics.NewUptimeSource()
	metricsEmitter := metrics.NewMetricsEmitter(logger, emitInterval, uptimeSource)

	policyPoller := &poller.Poller{
		Logger:       logger,
		PollInterval: pollInterval,
		SingleCycleFunc: (&converger.SinglePollCycle{
			Planners: []converger.Planner{
				dynamicPlanner,
			},
			Enforcer:      ruleEnforcer,
			MetricsSender: metricsSender,
			Logger:        logger,
		}).DoCycle,
	}

	debugServerAddress := fmt.Sprintf("%s:%d", conf.DebugServerHost, conf.DebugServerPort)
	debugServer := createCustomDebugServer(debugServerAddress, reconfigurableSink, iptablesLoggingState)
	members := grouper.Members{
		{"metrics_emitter", metricsEmitter},
		{"policy_poller", policyPoller},
		{"debug-server", debugServer},
	}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))
	logger.Info("starting")
	err = <-monitor.Wait()
	if err != nil {
		die(logger, "ifrit monitor", err)
	}
}

const (
	DEBUG = "debug"
	INFO  = "info"
	ERROR = "error"
	FATAL = "fatal"
)

func initLoggerSink(logger lager.Logger, level string) *lager.ReconfigurableSink {
	var logLevel lager.LogLevel
	switch strings.ToLower(level) {
	case DEBUG:
		logLevel = lager.DEBUG
	case INFO:
		logLevel = lager.INFO
	case ERROR:
		logLevel = lager.ERROR
	case FATAL:
		logLevel = lager.FATAL
	default:
		logLevel = lager.INFO
	}
	w := lager.NewWriterSink(os.Stdout, lager.DEBUG)
	return lager.NewReconfigurableSink(w, logLevel)
}

func createCustomDebugServer(listenAddress string, sink *lager.ReconfigurableSink, iptablesLoggingState *planner.LoggingState) ifrit.Runner {
	mux := debugserver.Handler(sink).(*http.ServeMux)
	mux.Handle("/iptables-c2c-logging", &handlers.IPTablesLogging{
		LoggingState: iptablesLoggingState,
	})
	return http_server.New(listenAddress, mux)
}
