package handlers

import (
	"log"
	"net/http"
	"os"
	"sync"
)

type FlapHandler struct {
	FlapInterval       int
	countUntilNextFlap int
	shouldFail         bool
	sync.RWMutex
}

func (h *FlapHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)

	if h.shouldFail {
		logger.Println("Flap handler failed")
		resp.WriteHeader(http.StatusInternalServerError)
	} else {
		logger.Println("Flap handler succeeded")
		resp.WriteHeader(http.StatusOK)
	}
	h.manageFlapping()
}

func (h *FlapHandler) manageFlapping() {
	h.RLock()
	defer h.RUnlock()
	if h.countUntilNextFlap == (h.FlapInterval - 1) {
		h.countUntilNextFlap = 0
		h.shouldFail = !h.shouldFail
	} else {
		h.countUntilNextFlap++
	}
}
