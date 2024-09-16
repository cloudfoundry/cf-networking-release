package handlers

import (
	"net/http"
	"strings"
)

type EchoSourceIPHandler struct{}

func (h *EchoSourceIPHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	parts := strings.Split(req.RemoteAddr, ":")
	// #nosec G104 - ignore error writing http response to avoid spamming logs on a DoS
	resp.Write([]byte(parts[0]))
}
