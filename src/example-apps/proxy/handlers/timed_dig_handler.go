package handlers

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"
)

type TimedDigHandler struct {
}

type TimedDigResponse struct {
	LookupTimeMS int64    `json:"lookup_time_ms"`
	IPs          []string `json:"ips"`
}

func (h *TimedDigHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	destination := strings.TrimPrefix(req.URL.Path, "/timed_dig/")
	destination = strings.Split(destination, ":")[0]

	start := time.Now()
	ips, err := net.LookupIP(destination)
	end := time.Now()
	if err != nil {
		handleDigError(err, destination, resp)
		return
	}

	lookupTime := end.Sub(start)

	var ip4s []string

	for _, ip := range ips {
		ip4s = append(ip4s, ip.To4().String())
	}

	responseBody, err := json.Marshal(TimedDigResponse{
		LookupTimeMS: lookupTime.Nanoseconds() / 1000000,
		IPs:          ip4s,
	})
	if err != nil {
		handleDigError(err, destination, resp)
		return
	}

	resp.Write(responseBody)
}
