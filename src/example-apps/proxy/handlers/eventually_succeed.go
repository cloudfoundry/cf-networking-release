package handlers

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

type EventuallySucceedHandler struct {
	callCount int
	sync.RWMutex
}

const succeedAfterDefault = 5

func (h *EventuallySucceedHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	cc := h.IncrementCallCount(logger)
	succeedAfter := getSucceedAfter()

	if cc > succeedAfter {
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

func getSucceedAfter() int {
	if v, ok := os.LookupEnv("EVENTUALLY_SUCCEED_AFTER_COUNT"); ok {
		count, err := strconv.Atoi(v)
		if err != nil {
			return succeedAfterDefault
		}
		return count
	}
	return succeedAfterDefault
}
