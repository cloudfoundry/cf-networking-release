package handlers

import (
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager"
)

type UptimeHandler struct {
	StartTime time.Time
}

func (h *UptimeHandler) ServeHTTP(logger lager.Logger, w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	currentTime := time.Now()
	uptime := currentTime.Sub(h.StartTime)
	w.Write([]byte(fmt.Sprintf("Network policy server, up for %v\n", uptime)))
	return
}
