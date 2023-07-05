package handlers

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

type EventuallyFailHandler struct {
	callCount int
	sync.RWMutex
}

const failAfterDefault = 5

func (h *EventuallyFailHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	cc := h.IncrementCallCount(logger)
	failAfter := getFailAfter()

	if cc > failAfter {
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

func getFailAfter() int {
	if v, ok := os.LookupEnv("EVENTUALLY_FAIL_AFTER_COUNT"); ok {
		count, err := strconv.Atoi(v)
		if err != nil {
			return failAfterDefault
		}
		return count
	}
	return failAfterDefault
}
