package sdcclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
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
	caPemBytes, err := ioutil.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("read CA file: %s", err)
	}
	caCertPool := x509.NewCertPool()
	if caCertPool.AppendCertsFromPEM(caPemBytes) != true {
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
		TLSClientConfig: tlsConfig,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
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

	for i := 0; i < 4; i++ {
		httpResp, err = s.client.Get(requestUrl)
		if err != nil {
			return []string{}, err
		}

		if httpResp.StatusCode == http.StatusOK {
			break
		} else {
			defer func(httpResp *http.Response) {
				io.Copy(ioutil.Discard, httpResp.Body)
				httpResp.Body.Close()
			}(httpResp)
		}
	}

	if httpResp.StatusCode != http.StatusOK {
		return []string{}, errors.New(fmt.Sprintf("Received non successful response from server: %+v", httpResp))
	}

	bytes, err := ioutil.ReadAll(httpResp.Body)
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
	ips := make([]string, numHosts, numHosts)
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
