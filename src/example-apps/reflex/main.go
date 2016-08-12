package main

import (
	"encoding/json"
	"errors"
	"example-apps/reflex/client"
	"example-apps/reflex/converger"
	"example-apps/reflex/handlers"
	"example-apps/reflex/pollers"
	"example-apps/reflex/store"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/localip"
)

type Timer struct{}

func (t *Timer) After() <-chan time.Time {
	return time.After(100 * time.Millisecond)
}

func dieWith(logger lager.Logger, action string, err error) {
	logger.Error(action, err)
	os.Exit(1)
}

func main() {
	logger := lager.NewLogger("reflex")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	systemPortString := os.Getenv("PORT")
	systemPort, err := strconv.Atoi(systemPortString)
	if err != nil {
		dieWith(logger, "get-port", errors.New("invalid required env var PORT"))
	}

	localIP, err := localip.LocalIP()
	if err != nil {
		dieWith(logger, "localip", err)
	}

	peerStore := store.New(localIP, 50, &sync.Mutex{})

	vcapAppBytes := []byte(os.Getenv("VCAP_APPLICATION"))
	var vcapApp struct {
		URIs []string
	}
	err = json.Unmarshal(vcapAppBytes, &vcapApp)
	if err != nil {
		dieWith(logger, "json unmarshal vcap app", err)
	}
	reflexClient := &client.ReflexClient{
		HttpClient: &http.Client{
			Timeout: 100 * time.Millisecond,
		},
		AppURL:  vcapApp.URIs[0],
		AppPort: systemPort,
	}

	peerConverger := &converger.Converger{
		Logger: logger,
		Client: reflexClient,
		Store:  peerStore,
	}

	peersHandler := &handlers.PeersHandler{
		Logger: logger.Session("peers"),
		Store:  peerStore,
	}
	httpServer := http_server.New(fmt.Sprintf("0.0.0.0:%d", systemPort), peersHandler)

	timer := &Timer{}
	peersPoller := &pollers.PeerPoller{
		Logger:    logger,
		Converger: peerConverger,
		Timer:     timer,
	}
	members := grouper.Members{
		{"http_server", httpServer},
		{"peers_poller", peersPoller},
	}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))
	logger.Info("starting")
	err = <-monitor.Wait()
	if err != nil {
		dieWith(logger, "ifrit-monitor", err)
	}
}
