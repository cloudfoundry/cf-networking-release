package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type SignalHandler struct{}

func (h *SignalHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	requestBytes := strings.TrimPrefix(req.URL.Path, "/signal/")
	signal, err := strconv.Atoi(requestBytes)
	if err != nil || signal < 0 {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("Signal must be a positive integer, got '%s'", requestBytes)))
		return
	}
	self, err := os.FindProcess(os.Getpid())
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("couldn't find process for pid %d", os.Getpid())))
		return
	}

	go func() {
		time.Sleep(1 * time.Second)
		self.Signal(syscall.Signal(signal))
	}()
	resp.Write([]byte(fmt.Sprintf("Ok, will signal %d in 1 second", signal)))
}
