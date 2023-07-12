package handlers

import (
	"log"
	"net/http"
	"os"
	"sync"
)

type EventuallySucceedHandler struct {
	SucceedAfterCount int
	callCount         int
	sync.RWMutex
}

func (h *EventuallySucceedHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	cc := h.IncrementCallCount(logger)

	if cc > h.SucceedAfterCount {
		logger.Println("EventuallySucceed handler succeeded")
		resp.WriteHeader(http.StatusOK)
	} else {
		logger.Println("EventuallySucceed handler hasn't succeeded yet")
		resp.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *EventuallySucceedHandler) IncrementCallCount(logger *log.Logger) int {
	h.RLock()
	defer h.RUnlock()
	h.callCount++
	return h.callCount
}
