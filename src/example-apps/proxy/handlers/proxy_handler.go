package handlers

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type ProxyHandler struct {
	Stats *Stats
}

var httpClient = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true,
		Dial: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 0,
		}).Dial,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}

func (h *ProxyHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	protocol := req.URL.Query().Get("protocol")
	if protocol == "" {
		protocol = "http"
	}
	destination := fmt.Sprintf("%s://%s", protocol, strings.TrimPrefix(req.URL.Path, "/proxy/"))

	before := time.Now()
	getResp, err := httpClient.Get(destination)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %s", err)
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("request failed: %s", err)))
		return
	}
	defer getResp.Body.Close()
	h.Stats.Add(time.Since(before).Seconds())

	readBytes, err := io.ReadAll(getResp.Body)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("read body failed: %s", err)))
		return
	}

	resp.Write(readBytes)
}
