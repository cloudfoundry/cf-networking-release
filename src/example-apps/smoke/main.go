package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

func launchServer(port int) {
	helloHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// #nosec G104 - ignore errors writing http responses
		w.Write([]byte("hello"))
	})

	internalAddressString := os.Getenv("CF_INSTANCE_INTERNAL_IP")
	internalAddress := net.ParseIP(internalAddressString)
	if internalAddress == nil {
		log.Fatal("invalid required env var CF_INSTANCE_INTERNAL_IP")
	}

	proxyURL := os.Getenv("PROXY_APP_URL")
	if proxyURL == "" {
		log.Fatal("invalid required env var PROXY_APP_URL")
	}

	selfProxyHandler := &SelfProxyHandler{
		SelfAddress: internalAddress.String(),
		ProxyURL:    proxyURL,
	}
	mux := http.NewServeMux()
	mux.Handle("/selfproxy", selfProxyHandler)
	mux.Handle("/", helloHandler)
	server := &http.Server{
		Addr:              fmt.Sprintf("0.0.0.0:%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	err := server.ListenAndServe()
	log.Printf("http server exited: %s\n", err)
}

func main() {
	systemPortString := os.Getenv("PORT")
	systemPort, err := strconv.Atoi(systemPortString)
	if err != nil {
		log.Fatal("invalid required env var PORT")
	}

	launchServer(systemPort)
}

type SelfProxyHandler struct {
	SelfAddress string
	ProxyURL    string
}

var httpClient = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true,
		Dial: (&net.Dialer{
			Timeout:   4 * time.Second,
			KeepAlive: 0,
		}).Dial,
	},
}

func (h *SelfProxyHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	newURL := fmt.Sprintf("%s/proxy/%s:8080", h.ProxyURL, h.SelfAddress)
	getResp, err := httpClient.Get(newURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %s", err)
		resp.WriteHeader(http.StatusInternalServerError)
		// #nosec G104 - ignore errors writing http responses
		resp.Write([]byte(fmt.Sprintf("request failed: %s", err)))
		return
	}
	defer getResp.Body.Close()

	resp.WriteHeader(getResp.StatusCode)
	switch getResp.StatusCode {
	case http.StatusOK:
		// #nosec G104 - ignore errors writing http responses
		resp.Write([]byte("OK"))
	default:
		// #nosec G104 - ignore errors writing http responses
		resp.Write([]byte("FAILED"))
	}
}
