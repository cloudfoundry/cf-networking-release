package lagerlevel

import (
	"code.cloudfoundry.org/lager/v3"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type Server struct {
	address string
	port    int
	sink    *lager.ReconfigurableSink
	logger  lager.Logger
}

func NewServer(address string, port int, sink *lager.ReconfigurableSink, logger lager.Logger) *Server {
	return &Server{
		address: address,
		port:    port,
		sink:    sink,
		logger:  logger,
	}
}

func (s *Server) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/log-level", s.handleRequest)

	address := fmt.Sprintf("%s:%d", s.address, s.port)
	httpServer := &http.Server{
		Addr:    address,
		Handler: mux,
	}
	httpServer.SetKeepAlivesEnabled(false)

	exited := make(chan error)
	go func() {
		err := httpServer.ListenAndServe()
		if err != nil {
			s.logger.Error("Listen and serve exited with error:", err)
			exited <- err
		}
	}()

	timeOut := time.Now().Add(5 * time.Second)

	url := fmt.Sprintf("http://%s", address)
	client := &http.Client{}

	for {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == 404 {
			break
		}
		if time.Now().After(timeOut) {
			httpServer.Close()
			return errors.New("failed to successfully connect to http server")
		}
		time.Sleep(100 * time.Millisecond)
	}

	s.logger.Info("server-started")

	close(ready)

	for {
		select {
		case err := <-exited:
			httpServer.Close()
			return err
		case <-signals:
			httpServer.Close()
			return nil
		}
	}
}

func (s *Server) handleRequest(resp http.ResponseWriter, req *http.Request) {
	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		s.logger.Info("Unable to read request body")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	body := string(bytes)

	var returnStatus int

	switch body {
	case "info":
		s.sink.SetMinLevel(lager.INFO)
		s.logger.Info("Log level set to INFO")
		returnStatus = http.StatusNoContent
	case "debug":
		s.sink.SetMinLevel(lager.DEBUG)
		s.logger.Info("Log level set to DEBUG")
		returnStatus = http.StatusNoContent
	default:
		s.logger.Info(fmt.Sprintf("Invalid log level requested: `%s`. Skipping.", body))
		returnStatus = http.StatusBadRequest
	}

	resp.WriteHeader(returnStatus)
}
