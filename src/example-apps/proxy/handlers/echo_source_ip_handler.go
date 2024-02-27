package handlers

import (
	"net/http"
	"strings"
)

type EchoSourceIPHandler struct{}

func (h *EchoSourceIPHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	parts := strings.Split(req.RemoteAddr, ":")
	resp.Write([]byte(parts[0]))
}
