package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"
	"strings"
)

type DigUDPHandler struct {
}

func (h *DigUDPHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	destination := strings.TrimPrefix(req.URL.Path, "/digudp/")
	destination = strings.Split(destination, ":")[0]

	cmd := exec.Command("dig", "+short", "+notcp", "+ignore", "@8.8.8.8", destination)
	output, err := cmd.Output()
	if err != nil {
		handleDigError(err, destination, resp)
		return
	}

	ips := strings.Split(string(output), "\n")
	ips = ips[0 : len(ips)-1]

	ipJson, err := json.Marshal(ips)
	if err != nil {
		handleDigError(err, destination, resp)
		return
	}

	if len(ips) == 0 {
		handleDigError(errors.New("no ips found"), destination, resp)
		return
	}

	resp.Write(ipJson)
}
