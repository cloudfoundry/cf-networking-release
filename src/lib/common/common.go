package common

import (
	"crypto/tls"
	"fmt"
	"policy-server/server_metrics"
	"policy-server/store"
	"time"

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
