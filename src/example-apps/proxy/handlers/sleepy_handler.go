package handlers

import (
	"net/http"
	"time"
)

type SleepyHandler struct {
	Port           int
	SleepyInterval int
}

func (h *SleepyHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	time.Sleep(time.Duration(h.SleepyInterval) * time.Second)

	respBytes := []byte("😴")
	resp.Write(respBytes)
}
