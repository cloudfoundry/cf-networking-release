package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
	"syscall"
	"io/ioutil"
	"strconv"
	"net"
	neturl "net/url"
	"errors"
	"strings"
	"code.cloudfoundry.org/lager"
	"os"
)

var logger lager.Logger

func main() {
	if err := mainWithError(); err != nil {
		log.Fatalf("silk-daemon-shutdown: %s", err)
	}
}

func mainWithError() error {
	repUrl := flag.String("repUrl", "", "path to rep url")
	repTimeout := flag.Int("repTimeout", 5, "timeout (seconds) between calls to rep")
	silkDaemonUrl := flag.String("silkDaemonUrl", "", "path to silk daemon url")
	silkDaemonTimeout := flag.Int("silkDaemonTimeout", 2, "timeout (seconds) between calls to silk daemon")
	silkDaemonPidPath := flag.String("silkDaemonPidPath", "", "pid file of silk daemon")
	pingServerTimeout := flag.Int("pingServerTimeout", 300, "timeout (seconds) when pinging if server is up")

	flag.Parse()

	logger = lager.NewLogger(fmt.Sprintf("%s.silk-daemon-shutdown", "cfnetworking"))
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	var err error
	repMaxAttempts := 40
	isRepUp, err := waitForServer("rep", *repUrl, *repTimeout, repMaxAttempts, *pingServerTimeout)
	if err != nil {
		return err
	}

	if isRepUp {
		logger.Debug(fmt.Sprintf("rep did not exit after %d ping attempts. Continuing", repMaxAttempts))
	}

	pidFileConents, err := ioutil.ReadFile(*silkDaemonPidPath)
	if err != nil {
		return err
	}

	pid, err := strconv.Atoi(strings.Trim(string(pidFileConents), "\n"))
	if err != nil {
		return err
	}

	logger.Debug(fmt.Sprintf("sending TERM signal to silk-daemon"))
	_ = syscall.Kill(pid, syscall.SIGTERM)

	silkDaemonMaxAttempts := 5
	silkDaemonIsUp, err := waitForServer("silk daemon", *silkDaemonUrl, *silkDaemonTimeout, silkDaemonMaxAttempts, *pingServerTimeout)
	if err != nil {
		return err
	}
	if silkDaemonIsUp {
		return errors.New(fmt.Sprintf("Silk Daemon Server did not exit after %d ping attempts", silkDaemonMaxAttempts))
	}

	return nil
}

func waitForServer(serverName string, serverUrl string, pollingTimeInSeconds int, maxAttempts int, pingTimeout int) (isServerUp bool, err error) {
	_, err = neturl.ParseRequestURI(serverUrl)
	if err != nil {
		return true, err
	}
	currentAttempt := 0

	for currentAttempt < maxAttempts {
		logger.Debug(fmt.Sprintf("waiting for the %s to exit", serverName))

		select {
		case <-time.After(time.Duration(pollingTimeInSeconds) * time.Second):
			if !checkIfServerUp(serverName, serverUrl) {
				return false, nil
			}
			currentAttempt++
		case <-time.After(time.Duration(pingTimeout) * time.Second):
			return true, nil
		}
	}

	return true, nil
}

func checkIfServerUp(serverName string, url string) bool {
	httpClient := &http.Client{
		Transport: &http.Transport{},
		Timeout:   5 * time.Second,
	}

	logger.Debug(fmt.Sprintf("pinging %s", url))
	response, err := httpClient.Get(url)

	if err != nil {
		if netErr, ok := err.(net.Error); ok {
			if netErr.Timeout() {
				logger.Debug("pinging server timed out. trying again.")
				return true

			}
			if netErr.Temporary() {
				logger.Debug("pinging server returned temporary error. trying again.")
				return true
			}
		}
	} else {
		defer response.Body.Close()
		if response.StatusCode >= http.StatusOK && response.StatusCode <= http.StatusPartialContent {
			return true
		}
	}

	logger.Debug(fmt.Sprintf("could not ping %s server. Server is down", serverName))
	return false
}
