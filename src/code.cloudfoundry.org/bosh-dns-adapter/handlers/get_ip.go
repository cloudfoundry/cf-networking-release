package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"code.cloudfoundry.org/lager/v3"
	"golang.org/x/net/dns/dnsmessage"
)

//go:generate counterfeiter -o fakes/sdc_client.go --fake-name SDCClient . sdcClient
type sdcClient interface {
	IPs(hostname string) ([]string, error)
}

//go:generate counterfeiter -o fakes/metrics_sender.go --fake-name MetricsSender . metricsSender
type metricsSender interface {
	IncrementCounter(name string)
	SendDuration(name string, duration time.Duration)
}

type GetIP struct {
	SDCClient                  sdcClient
	InternalServiceMeshDomains []string
	Logger                     lager.Logger
	MetricsSender              metricsSender
}

func (g GetIP) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if isHealthCheck(req) {
		w.WriteHeader(200)
		return
	}
	name := getQueryParam(req, "name", "")
	dnsType := getQueryParam(req, "type", "1")

	requestLogger := g.Logger.Session("serve-request")

	if dnsType != "1" {
		g.writeResponse(w, dnsmessage.RCodeSuccess, name, dnsType, nil)
		requestLogger.Debug("unsupported record type", lager.Data{
			"ips":          "",
			"service-name": name,
		})
		return
	}

	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		g.writeResponse(w, dnsmessage.RCodeServerFailure, name, dnsType, nil)
		requestLogger.Debug("name parameter empty", lager.Data{
			"ips":          "",
			"service-name": "",
		})
		return
	}

	var ips []string
	var err error
	var componentErrorMsg string
	start := time.Now()
	ips, err = g.SDCClient.IPs(name)
	componentErrorMsg = "could not connect to service discovery controller"
	duration := time.Since(start).Nanoseconds()

	if err != nil {
		g.writeErrorResponse(w, err)
		requestLogger.Error(componentErrorMsg,
			err,
			lager.Data{
				"ips":          "",
				"service-name": name,
			})

		g.MetricsSender.IncrementCounter("DNSRequestFailures")
		return
	}

	g.writeResponse(w, dnsmessage.RCodeSuccess, name, dnsType, ips)
	requestLogger.Debug("success", lager.Data{
		"ips":          strings.Join(ips, ","),
		"service-name": name,
		"duration-ns":  duration,
	})
}

func (g GetIP) writeErrorResponse(resp http.ResponseWriter, err error) {
	resp.WriteHeader(http.StatusInternalServerError)
	_, err = resp.Write([]byte(err.Error()))
	if err != nil {
		g.Logger.Error("Error writing to http response body", err)
	}
}

func isHealthCheck(req *http.Request) bool {
	return req.URL.Path == "/health"
}

func getQueryParam(req *http.Request, key, defaultValue string) string {
	queryValue := req.URL.Query().Get(key)
	if queryValue == "" {
		return defaultValue
	}

	return queryValue
}

type Answer struct {
	Name   string `json:"name"`
	RRType uint16 `json:"type"`
	TTL    uint32 `json:"TTL"`
	Data   string `json:"data"`
}

func (g GetIP) writeResponse(resp http.ResponseWriter, dnsResponseStatus dnsmessage.RCode, requestedInfraName string, dnsType string, ips []string) {
	responseBody, err := buildResponseBody(dnsResponseStatus, requestedInfraName, dnsType, ips)
	if err != nil {
		g.Logger.Error("Error building response", err)
		return
	}

	_, err = resp.Write([]byte(responseBody))
	if err != nil {
		g.Logger.Error("Error writing to http response body", err)
	}

	g.Logger.Debug("HTTPServer access")
}

func buildResponseBody(dnsResponseStatus dnsmessage.RCode, requestedInfraName string, dnsType string, ips []string) (string, error) {
	answers := make([]Answer, len(ips))
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
