package routes

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"service-discovery-controller/config"

	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/middleware"
	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/paraphernalia/secure/tlsconfig"

	"time"
)

type Server struct {
	config             *config.Config
	logger             lager.Logger
	addressTable       AddressTable
	dnsRequestRecorder DNSRequestRecorder
	metricsSender      MetricsSender
}

type host struct {
	IPAddress       string                 `json:"ip_address"`
	LastCheckIn     string                 `json:"last_check_in"`
	Port            int32                  `json:"port"`
	Revision        string                 `json:"revision"`
	Service         string                 `json:"service"`
	ServiceRepoName string                 `json:"service_repo_name"`
	Tags            map[string]interface{} `json:"tags"`
}

type registration struct {
	Hosts   []host `json:"hosts"`
	Env     string `json:"env"`
	Service string `json:"service"`
}

type routes struct {
	Addresses []address `json:"addresses"`
}

type address struct {
	Hostname string   `json:"hostname"`
	Ips      []string `json:"ips"`
}

//go:generate counterfeiter -o fakes/address_table.go --fake-name AddressTable . AddressTable
type AddressTable interface {
	Lookup(hostname string) []string
	GetAllAddresses() map[string][]string
	IsWarm() bool
}

//go:generate counterfeiter -o fakes/metrics_sender.go --fake-name MetricsSender . MetricsSender
type MetricsSender interface {
	SendDuration(string, time.Duration)
}

//go:generate counterfeiter -o fakes/dns_request_recorder.go --fake-name DNSRequestRecorder . DNSRequestRecorder
type DNSRequestRecorder interface {
	RecordRequest()
}

func NewServer(addressTable AddressTable, config *config.Config, dnsRequestRecorder DNSRequestRecorder, metricsSender MetricsSender, logger lager.Logger) *Server {
	return &Server{
		addressTable:       addressTable,
		config:             config,
		dnsRequestRecorder: dnsRequestRecorder,
		metricsSender:      metricsSender,
		logger:             logger,
	}
}

func (s *Server) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	mux := http.NewServeMux()

	metricSender := metrics.MetricsSender{
		Logger: lager.NewLogger("bosh-dns-adapter"),
	}

	metricsWrap := func(name string, handler http.Handler) http.Handler {
		metricsWrapper := middleware.MetricWrapper{
			Name:          name,
			MetricsSender: &metricSender,
		}
		return metricsWrapper.Wrap(handler)
	}

	mux.HandleFunc("/v1/registration/", metricsWrap("Registration", http.HandlerFunc(s.handleRegistrationRequest)).ServeHTTP)
	mux.HandleFunc("/routes", s.handleRoutesRequest)

	tlsConfig, err := s.buildTLSServerConfig()
	if err != nil {
		return err
	}

	serverAddress := fmt.Sprintf("%s:%s", s.config.Address, s.config.Port)
	httpServer := &http.Server{
		Addr:      serverAddress,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	exited := make(chan error)
	go func() {
		serveErr := httpServer.ListenAndServeTLS("", "")
		s.logger.Info("server-exited")
		exited <- serveErr
	}()

	time.Sleep(time.Microsecond)
	close(ready)
	s.logger.Info("server-started")

	for {
		select {
		case err := <-exited:
			httpServer.Close()
			s.logger.Info(fmt.Sprintf("SDC http server exiting with: %v", err))
			return err
		case signal := <-signals:
			httpServer.Close()
			s.logger.Info(fmt.Sprintf("SDC http server exiting with signal: %v", signal))
			return nil
		}
	}
}

func (s *Server) buildTLSServerConfig() (*tls.Config, error) {
	caCert, err := ioutil.ReadFile(s.config.CACert)
	if err != nil {
		fmt.Errorf("unable to read ca file: %s", err)
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(s.config.ServerCert, s.config.ServerKey)
	if err != nil {
		fmt.Errorf("unable to load x509 key pair: %s", err)
		return nil, err
	}

	tlsConfig := tlsconfig.Build(
		tlsconfig.WithIdentity(cert),
		tlsconfig.WithInternalServiceDefaults(),
	)

	serverConfig := tlsConfig.Server(tlsconfig.WithClientAuthentication(caCertPool))
	serverConfig.BuildNameToCertificate()
	return serverConfig, err
}

func (s *Server) handleRegistrationRequest(resp http.ResponseWriter, req *http.Request) {
	serviceKey := path.Base(req.URL.Path)

	isWarm := s.addressTable.IsWarm()
	if !isWarm {
		http.Error(resp, "address table is not warm", http.StatusInternalServerError)
		s.logger.Debug("failed-request", lager.Data{
			"serviceKey": serviceKey,
			"reason":     "address-table-not-warm",
		})
		return
	}

	lookupStartTime := time.Now()
	ips := s.addressTable.Lookup(serviceKey)
	lookupDuration := time.Now().Sub(lookupStartTime)
	s.metricsSender.SendDuration("addressTableLookupTime", lookupDuration)
	hosts := make([]host, cap(ips))
	for index, ip := range ips {
		hosts[index] = host{
			IPAddress: ip,
			Tags:      make(map[string]interface{}),
		}
	}

	var err error
	json, err := json.Marshal(registration{Hosts: hosts})
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = resp.Write(json)
	if err != nil {
		s.logger.Debug("Error writing to http response body")
	}

	s.dnsRequestRecorder.RecordRequest()

	s.logger.Debug("HTTPServer access", lager.Data(map[string]interface{}{
		"serviceKey":   serviceKey,
		"responseJson": string(json),
	}))
}

func (s *Server) handleRoutesRequest(resp http.ResponseWriter, req *http.Request) {
	availableAddresses := s.addressTable.GetAllAddresses()
	addresses := []address{}
	for i, availableAddress := range availableAddresses {
		addresses = append(addresses, address{
			Hostname: i,
			Ips:      availableAddress,
		})
	}

	var err error
	json, err := json.Marshal(routes{Addresses: addresses})
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = resp.Write(json)
	if err != nil {
		s.logger.Debug("Error writing to http response body")
	}

	s.logger.Debug("HTTPServer access", lager.Data(map[string]interface{}{
		"responseJson": string(json),
	}))
}
