package common

import (
	"crypto/tls"
	"fmt"
	"os"
	"policy-server/server_metrics"
	"policy-server/store"
	"strings"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/rata"
)

const (
	DEBUG        = "debug"
	INFO         = "info"
	ERROR        = "error"
	FATAL        = "fatal"
	emitInterval = 30 * time.Second
)

func InitLoggerSink(logger lager.Logger, level string) *lager.ReconfigurableSink {
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

func InitMetricsEmitter(logger lager.Logger, wrappedStore *store.MetricsWrapper) *metrics.MetricsEmitter {
	totalPoliciesSource := server_metrics.NewTotalPoliciesSource(wrappedStore)
	uptimeSource := metrics.NewUptimeSource()
	return metrics.NewMetricsEmitter(logger, emitInterval, uptimeSource, totalPoliciesSource)
}

func InitServer(logger lager.Logger, tlsConfig *tls.Config, host string, port int, handlers rata.Handlers, routes rata.Routes) ifrit.Runner {
	router, err := rata.NewRouter(routes, handlers)
	if err != nil {
		logger.Fatal("create-rata-router", err) // not tested
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	if tlsConfig != nil {
		return http_server.NewTLSServer(addr, router, tlsConfig)
	}
	return http_server.New(addr, router)
}
