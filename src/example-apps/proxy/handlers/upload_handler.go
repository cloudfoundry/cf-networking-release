package handlers

import (
	"fmt"
	"io"
	"net/http"
)

type UploadHandler struct{}

func (h *UploadHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		// #nosec G104 - ignore error writing http response to avoid spamming logs on a DoS
		resp.Write([]byte("0 bytes received and read"))
		return
	}
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		// #nosec G104 - ignore error writing http response to avoid spamming logs on a DoS
		resp.Write([]byte(fmt.Sprintf("error: %s", err)))
		return
	}

	// #nosec G104 - ignore error writing http response to avoid spamming logs on a DoS
	resp.Write([]byte(fmt.Sprintf("%d bytes received and read", len(bodyBytes))))
}
