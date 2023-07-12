package handlers

import (
	"log"
	"net/http"
	"os"
	"sync"
)

type EventuallyFailHandler struct {
	FailAfterCount int
	callCount      int
	sync.RWMutex
}

func (h *EventuallyFailHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	cc := h.IncrementCallCount(logger)

	if cc > h.FailAfterCount {
		logger.Println("EventuallyFail handler failed")
		resp.WriteHeader(http.StatusInternalServerError)
	} else {
		logger.Println("EventuallyFail handler hasn't failed yet")
		resp.WriteHeader(http.StatusOK)
	}
}

func (h *EventuallyFailHandler) IncrementCallCount(logger *log.Logger) int {
	h.RLock()
	defer h.RUnlock()
	h.callCount++
	return h.callCount
}
