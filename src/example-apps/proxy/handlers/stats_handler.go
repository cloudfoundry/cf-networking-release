package handlers

import (
	"encoding/json"
	"net/http"
	"sync"
)

type Stats struct {
	// Locker  sync.Locker
	Latency []int `json:"latency"`
	sync.RWMutex
}

func (s *Stats) Add(latency int) {
	s.Lock()
	defer s.Unlock()
	s.Latency = append(s.Latency, latency)
}

func (s *Stats) Clear() {
	s.Lock()
	defer s.Unlock()
	s.Latency = []int{}
}

func (s *Stats) GetLatency() []int {
	s.RLock()
	defer s.RUnlock()
	return s.Latency
}

type StatsHandler struct {
	Stats *Stats
}

func (h *StatsHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.Method == "DELETE" {
		h.Stats.Clear()
		return
	}

	respBytes, err := json.Marshal(h.Stats)
	if err != nil {
		panic(err)
	}
	resp.Write(respBytes)
	return
}
