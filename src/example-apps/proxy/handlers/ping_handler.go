package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type PingHandler struct {
}

func handleError(err error, destination string, resp http.ResponseWriter) {
	msg := fmt.Sprintf("Ping failed to destination: %s: %s", destination, err)
	fmt.Fprint(os.Stderr, msg)
	resp.WriteHeader(http.StatusInternalServerError)
	// #nosec G104 - ignore error writing http response to avoid spamming logs on a DoS
	resp.Write([]byte(msg))
}

func (h *PingHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	destination := strings.TrimPrefix(req.URL.Path, "/ping/")
	destination = strings.Split(destination, ":")[0]

	pingPath := "/bin/ping"
	_, err := os.Stat(pingPath)
	if err != nil {
		pingPath = "/sbin/ping"
	}
	cmd := exec.Command(pingPath, "-c", "1", destination)
	err = cmd.Start()
	if err != nil {
		handleError(err, destination, resp)
		return
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(10 * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			handleError(fmt.Errorf("error killing hung ping: %s", err), destination, resp)
			return
		}
		handleError(errors.New("killing ping after timed out"), destination, resp)
		return

	case err := <-done:
		if err != nil {
			handleError(err, destination, resp)
			return
		}
	}

	// #nosec G104 - ignore error writing http response to avoid spamming logs on a DoS
	resp.Write([]byte(fmt.Sprintf("Ping succeeded to destination: %s", destination)))
}
