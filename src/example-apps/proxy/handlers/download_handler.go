package handlers

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type DownloadHandler struct{}

func (h *DownloadHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	requestBytes := strings.TrimPrefix(req.URL.Path, "/download/")
	numBytes, err := strconv.Atoi(requestBytes)
	if err != nil || numBytes < 0 {
		resp.WriteHeader(http.StatusInternalServerError)
		// #nosec G104 - ignore error writing http response to avoid spamming logs on a DoS
		resp.Write([]byte(fmt.Sprintf("requested number of bytes must be a positive integer, got: %s", requestBytes)))
		return
	}

	respBytes := make([]byte, numBytes)
	_, err = rand.Read(respBytes)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		// #nosec G104 - ignore error writing http response to avoid spamming logs on a DoS
		resp.Write([]byte(fmt.Sprintf("unable to generate random bytes for your download: %s", err)))
	}
	// #nosec G104 - ignore error writing http response to avoid spamming logs on a DoS
	resp.Write(respBytes)
}
