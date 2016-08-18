package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type ProxyHandler struct{}

func (h *ProxyHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	destination := strings.TrimPrefix(req.URL.Path, "/proxy/")
	destination = "http://" + destination
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}
	getResp, err := client.Get(destination)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %s", err)
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("request failed: %s", err)))
		return
	}
	defer getResp.Body.Close()

	readBytes, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("read body failed: %s", err)))
		return
	}

	resp.Write(readBytes)
}
