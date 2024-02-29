package sdcclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type ServiceDiscoveryClient struct {
	serverURL string
	client    *http.Client
}

type serverResponse struct {
	Hosts []host `json:"Hosts"`
}

type host struct {
	IPAddress string `json:"ip_address"`
}

func NewServiceDiscoveryClient(serverURL, caPath, clientCertPath, clientKeyPath string) (*ServiceDiscoveryClient, error) {
	caPemBytes, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("read CA file: %s", err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caPemBytes) {
		return nil, fmt.Errorf("load CA file into cert pool")
	}

	cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load client key pair: %s", err)
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		ClientCAs:    caCertPool,
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
	}

	tlsConfig.BuildNameToCertificate()

	tr := &http.Transport{
		TLSClientConfig:     tlsConfig,
		IdleConnTimeout:     0,
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 50,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * 10,
	}

	return &ServiceDiscoveryClient{
		serverURL: serverURL,
		client:    client,
	}, nil
}

func (s *ServiceDiscoveryClient) IPs(infrastructureName string) ([]string, error) {
	requestUrl := fmt.Sprintf("%s/v1/registration/%s", s.serverURL, infrastructureName)

	var (
		err      error
		httpResp *http.Response
	)

	retries := 3
	for i := 0; i <= retries; i++ {
		httpResp, err = s.client.Get(requestUrl)
		if err != nil {
			return []string{}, err
		}

		if httpResp.StatusCode == http.StatusOK {
			break
		}

		defer func(httpResp *http.Response) {
			io.Copy(io.Discard, httpResp.Body)
			httpResp.Body.Close()
		}(httpResp)

		if i != retries {
			jitter := rand.Intn(75) + 50   // jitter between 50ms and 125ms
			interval := jitter + (500 * i) // interval will be 0/500/1000 + jitter
			time.Sleep(time.Duration(interval) * time.Millisecond)
		}
	}

	if httpResp.StatusCode != http.StatusOK {
		return []string{}, fmt.Errorf("Received non successful response from server: %+v", httpResp)
	}

	bytes, err := io.ReadAll(httpResp.Body)
	httpResp.Body.Close()
	if err != nil {
		return []string{}, err
	}

	var serverResponse *serverResponse
	err = json.Unmarshal(bytes, &serverResponse)
	if err != nil {
		return []string{}, err
	}

	numHosts := len(serverResponse.Hosts)
	ips := make([]string, numHosts)
	for i, host := range serverResponse.Hosts {
		ips[i] = host.IPAddress
	}

	shuffle(ips)

	return ips, nil
}

func shuffle(vals []string) {
	r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	for len(vals) > 0 {
		n := len(vals)
		randIndex := r.Intn(n)
		vals[n-1], vals[randIndex] = vals[randIndex], vals[n-1]
		vals = vals[:n-1]
	}
}
