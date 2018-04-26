package main

import (
	"bosh-dns-adapter/config"
	"bosh-dns-adapter/sdcclient"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"time"

	"code.cloudfoundry.org/cf-networking-helpers/lagerlevel"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/middleware"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/dropsonde"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
	"golang.org/x/net/dns/dnsmessage"
)

func main() {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM, os.Interrupt)

	logger := lager.NewLogger("bosh-dns-adapter")
	writerSink := lager.NewWriterSink(os.Stdout, lager.DEBUG)
	sink := lager.NewReconfigurableSink(writerSink, lager.INFO)
	logger.RegisterSink(sink)
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))

	configPath := flag.String("c", "", "path to config file")
	flag.Parse()

	bytes, err := ioutil.ReadFile(*configPath)
	if err != nil {
		logger.Info("Could not read config file", lager.Data{"path": *configPath})
		os.Exit(2)
	}

	config, err := config.NewConfig(bytes)
	if err != nil {
		logger.Info("Could not parse config file", lager.Data{"path": *configPath})
		os.Exit(2)
	}

	address := fmt.Sprintf("%s:%s", config.Address, config.Port)
	l, err := net.Listen("tcp", address)
	if err != nil {
		logger.Error(fmt.Sprintf("Address (%s) not available", address), err)
		os.Exit(1)
	}

	sdcServerUrl := fmt.Sprintf("https://%s:%s",
		config.ServiceDiscoveryControllerAddress,
		config.ServiceDiscoveryControllerPort,
	)

	metronAddress := fmt.Sprintf("127.0.0.1:%d", config.MetronPort)
	err = dropsonde.Initialize(metronAddress, "bosh-dns-adapter")
	if err != nil {
		logger.Error("Unable to initialize dropsonde", err, lager.Data{"metron_address": metronAddress})
		os.Exit(1)
	}

	sdcClient, err := sdcclient.NewServiceDiscoveryClient(sdcServerUrl, config.CACert, config.ClientCert, config.ClientKey)
	if err != nil {
		logger.Error("Unable to create service discovery client", err)
		os.Exit(1)
	}

	requestLogger := logger.Session("serve-request")

	metricSender := metrics.MetricsSender{
		Logger: logger.Session("bosh-dns-adapter"),
	}

	metricsWrap := func(name string, handler http.Handler) http.Handler {
		metricsWrapper := middleware.MetricWrapper{
			Name:          name,
			MetricsSender: &metricSender,
		}
		return metricsWrapper.Wrap(handler)
	}

	go func() {
		http.Serve(l, metricsWrap("GetIPs", http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			dnsType := getQueryParam(req, "type", "1")
			name := getQueryParam(req, "name", "")

			if dnsType != "1" {
				writeResponse(resp, dnsmessage.RCodeSuccess, name, dnsType, nil, logger)
				requestLogger.Debug("unsupported record type", lager.Data{
					"ips":          "",
					"service-name": name,
				})
				return
			}

			if name == "" {
				resp.WriteHeader(http.StatusBadRequest)
				writeResponse(resp, dnsmessage.RCodeServerFailure, name, dnsType, nil, logger)
				requestLogger.Debug("name parameter empty", lager.Data{
					"ips":          "",
					"service-name": "",
				})
				return
			}

			ips, err := sdcClient.IPs(name)
			if err != nil {
				wrappedErr := errors.New(fmt.Sprintf("Error querying Service Discover Controller: %s", err))
				writeErrorResponse(resp, wrappedErr, logger)
				requestLogger.Error("could not connect to service discovery controller",
					wrappedErr,
					lager.Data{
						"ips":          "",
						"service-name": name,
					})

				metricSender.IncrementCounter("DNSRequestFailures")
				return
			}

			writeResponse(resp, dnsmessage.RCodeSuccess, name, dnsType, ips, logger)
			requestLogger.Debug("success", lager.Data{
				"ips":          strings.Join(ips, ","),
				"service-name": name,
			})
		})))
	}()

	uptimeSource := metrics.NewUptimeSource()
	metricsEmitter := metrics.NewMetricsEmitter(
		lager.NewLogger("bosh-dns-adapter"),
		time.Duration(config.MetricsEmitSeconds)*time.Second,
		uptimeSource,
	)

	members := grouper.Members{
		{"metrics-emitter", metricsEmitter},
		{"log-level-server", lagerlevel.NewServer(config.LogLevelAddress, config.LogLevelPort, sink, logger.Session("log-level-server"))},
	}
	group := grouper.NewOrdered(os.Interrupt, members)
	monitor := ifrit.Invoke(sigmon.New(group))

	go func() {
		err = <-monitor.Wait()
		if err != nil {
			logger.Error("ifrit-failure", err)
			os.Exit(1)
		}
	}()

	logger.Info("server-started")
	select {
	case sig := <-signalChannel:
		monitor.Signal(sig)
		l.Close()
		logger.Info("server-stopped")
		return
	}
}
func getQueryParam(req *http.Request, key, defaultValue string) string {
	queryValue := req.URL.Query().Get(key)
	if queryValue == "" {
		return defaultValue
	}

	return queryValue
}

func writeErrorResponse(resp http.ResponseWriter, err error, logger lager.Logger) {
	resp.WriteHeader(http.StatusInternalServerError)
	_, err = resp.Write([]byte(err.Error()))
	if err != nil {
		logger.Error("Error writing to http response body", err)
	}
}

func writeResponse(resp http.ResponseWriter, dnsResponseStatus dnsmessage.RCode, requestedInfraName string, dnsType string, ips []string, logger lager.Logger) {
	responseBody, err := buildResponseBody(dnsResponseStatus, requestedInfraName, dnsType, ips)
	if err != nil {
		logger.Error("Error building response", err)
		return
	}

	_, err = resp.Write([]byte(responseBody))
	if err != nil {
		logger.Error("Error writing to http response body", err)
	}

	logger.Debug("HTTPServer access")
}

type Answer struct {
	Name   string `json:"name"`
	RRType uint16 `json:"type"`
	TTL    uint32 `json:"TTL"`
	Data   string `json:"data"`
}

func buildResponseBody(dnsResponseStatus dnsmessage.RCode, requestedInfraName string, dnsType string, ips []string) (string, error) {
	answers := make([]Answer, len(ips), len(ips))
	for i, ip := range ips {
		answers[i] = Answer{
			Name:   requestedInfraName,
			RRType: uint16(dnsmessage.TypeA),
			Data:   ip,
			TTL:    0,
		}
	}

	bytes, err := json.Marshal(answers)
	if err != nil {
		return "", err // not tested
	}

	template := `{
		"Status": %d,
		"TC": false,
		"RD": false,
		"RA": false,
		"AD": false,
		"CD": false,
		"Question":
		[
			{
				"name": "%s",
				"type": %s
			}
		],
		"Answer": %s,
		"Additional": [ ],
		"edns_client_subnet": "0.0.0.0/0"
	}`

	return fmt.Sprintf(template, dnsResponseStatus, requestedInfraName, dnsType, string(bytes)), nil
}
