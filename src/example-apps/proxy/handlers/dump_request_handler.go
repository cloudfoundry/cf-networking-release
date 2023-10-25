package handlers

import (
	"net/http"
)

type DumpRequestHandler struct {
}

func (h *DumpRequestHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	reqBytes, _ := httputil.DumpRequest(req, false)

	if returnHeadersParam := req.URL.Query().Get("returnHeaders"); returnHeadersParam == "true" {
		originalHeaders := req.Header.Clone()
		for name, values := range originalHeaders {
			for _, v := range values {
				resp.Header().Add(name, v)
			}
		}

		// if X-Proxy-Settable-Debug-Header not in original request header, set default value
		if h := originalHeaders.Get("X-Proxy-Settable-Debug-Header"); h == "" {
			resp.Header().Set("X-Proxy-Settable-Debug-Header", "default-settable-value-from-within-proxy-src-code")
		}

		// We are going to explicitly set 'X-Proxy-Immutable-Debug-Header' at the end so it's immutable
		resp.Header().Set("X-Proxy-Immutable-Debug-Header", "default-immutable-value-from-within-proxy-src-code")
	}

	resp.Write(reqBytes)
}
