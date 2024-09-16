package handlers

import (
	"fmt"
	"net/http"
	"time"
)

type UptimeHandler struct {
	StartTime time.Time
}

func (h *UptimeHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	currentTime := time.Now()
	uptime := currentTime.Sub(h.StartTime)
	// #nosec G104 - ignore errors writing http responses to avoid spamming logs during a DoS
	w.Write([]byte(fmt.Sprintf("Network policy server, up for %v\n", uptime)))
}
