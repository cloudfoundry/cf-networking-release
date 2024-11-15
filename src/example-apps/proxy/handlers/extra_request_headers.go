package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type ExtraRequestHeadersHandler struct {
	Stats *Stats
}

func (h *ExtraRequestHeadersHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	protocol := req.URL.Query().Get("protocol")
	if protocol == "" {
		protocol = "http"
	}

	extraHeaders := req.Header.Get("NumExtraRequestHeaders")
	numExtraHeaders, err := strconv.Atoi(extraHeaders)
	if err != nil {
		logger.Println("meow")
		resp.WriteHeader(http.StatusBadRequest)
	}

	destination := fmt.Sprintf("%s://%s", protocol, strings.TrimPrefix(req.URL.Path, "/extrarequestheaders/"))

	getReq, err := http.NewRequest("GET", destination, nil)
	if err != nil {
		logger.Println("meow")
		resp.WriteHeader(http.StatusBadRequest)
	}

	var header string
	for i := 0; i < numExtraHeaders; i++ {
		header = fmt.Sprintf("meow-%d", i)
		getReq.Header.Set(header, header)
	}

	getResp, err := httpClient.Do(getReq)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %s", err)
		resp.WriteHeader(http.StatusInternalServerError)
		// #nosec G104 - ignore error writing http response to avoid spamming logs on a DoS
		resp.Write([]byte(fmt.Sprintf("request failed: %s", err)))
		return
	}
	defer getResp.Body.Close()

	readBytes, err := io.ReadAll(getResp.Body)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		// #nosec G104 - ignore error writing http response to avoid spamming logs on a DoS
		resp.Write([]byte(fmt.Sprintf("read body failed: %s", err)))
		return
	}

	// #nosec G104 - ignore error writing http response to avoid spamming logs on a DoS
	resp.Write(readBytes)
}
