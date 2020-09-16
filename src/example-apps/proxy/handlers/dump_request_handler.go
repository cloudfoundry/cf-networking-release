package handlers

import (
	"net/http"
	"net/http/httputil"
)

type DumpRequestHandler struct {
}

func (h *DumpRequestHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	reqBytes, _ := httputil.DumpRequest(req, false)
	resp.Write(reqBytes)
}
