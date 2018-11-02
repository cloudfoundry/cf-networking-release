package common

import (
	"crypto/tls"
	"fmt"
	"policy-server/server_metrics"
	"policy-server/store"
	"time"

	"code.cloudfoundry.org/bbs/db/sqldb/helpers/monitor"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"
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

func GetLagerConfig() lagerflags.LagerConfig {
	lagerConfig := lagerflags.DefaultLagerConfig()
	lagerConfig.TimeFormat = lagerflags.FormatRFC3339
	return lagerConfig
}

func InitMetricsEmitter(logger lager.Logger, wrappedStore *store.MetricsWrapper, db metrics.Db, monitor monitor.Monitor) *metrics.MetricsEmitter {
	metricSources := []metrics.MetricSource{
		metrics.NewUptimeSource(),
		server_metrics.NewTotalPoliciesSource(wrappedStore),
	}
	metricSources = append(metricSources, metrics.NewDBMonitorSource(db, monitor)...)
	return metrics.NewMetricsEmitter(logger, emitInterval, metricSources...)
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
