package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"route_populator/publisher"
	"route_populator/runner"
	"runtime"
	"syscall"
	"time"

	"github.com/nats-io/nats"
)

var natsEndpoint = flag.String(
	"nats",
	"",
	"Endpoint of the NATS server (including scheme, credentials, and port)",
)

var backendHost = flag.String(
	"backendHost",
	"",
	"Host for the destination of the route.",
)

var backendPort = flag.Int(
	"backendPort",
	0,
	"Port for the destination of the route.",
)

var appDomain = flag.String(
	"appDomain",
	"",
	"The domain name for the routes to register.",
)

var appName = flag.String(
	"appName",
	"",
	"The name of the app for the route to register.",
)

var numRoutes = flag.Int(
	"numRoutes",
	0,
	"Number of routes to populate the routing table with.",
)

var heartbeatInterval = flag.Int(
	"heartbeatInterval",
	60,
	"Time (in seconds) between sending routes.",
)

var publishDelayString = flag.String(
	"publishDelay",
	"50us",
	"Time to wait (duration string) between each publishing of a NATS message",
)

var publishDelay time.Duration

func main() {
	checkRequiredFields()
	startPopulatingRoutes()
}

func checkRequiredFields() {
	flag.Parse()
	checkFailed := false

	if *natsEndpoint == "" {
		fmt.Fprintf(os.Stderr, "-nats must be provided\n")
		checkFailed = true
	}

	if *backendHost == "" {
		fmt.Fprintf(os.Stderr, "-backendHost must be provided\n")
		checkFailed = true
	}

	if *backendPort <= 0 {
		fmt.Fprintf(os.Stderr, "-backendPort must be provided\n")
		checkFailed = true
	}

	if *appDomain == "" {
		fmt.Fprintf(os.Stderr, "-appDomain must be provided\n")
		checkFailed = true
	}

	if *appName == "" {
		fmt.Fprintf(os.Stderr, "-appName must be provided\n")
		checkFailed = true
	}

	if *numRoutes <= 0 {
		fmt.Fprintf(os.Stderr, "-numRoutes must be provided\n")
		checkFailed = true
	}

	if *heartbeatInterval <= 0 {
		fmt.Fprintf(os.Stderr, "-heartbeatInterval must be greater than 0\n")
		checkFailed = true
	}

	var err error
	publishDelay, err = time.ParseDuration(*publishDelayString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "-publishDelay is an invalid string: %s\n", err)
		checkFailed = true
	}

	if checkFailed {
		fmt.Fprintf(os.Stderr, "\n")
		flag.Usage()
		os.Exit(1)
	}
}

func startPopulatingRoutes() {
	job := publisher.Job{
		PublishingEndpoint: *natsEndpoint,

		BackendHost: *backendHost,
		BackendPort: *backendPort,

		AppDomain: *appDomain,
		AppName:   *appName,

		StartRange: 0,
		EndRange:   *numRoutes,
	}

	numCPU := runtime.NumCPU()
	// Heuristic to avoid spawning more goroutines than needed
	if *numRoutes < 1000 {
		numCPU = 1
	}
	interval := time.Duration(*heartbeatInterval) * time.Second
	r := runner.NewRunner(createNATSConnection, job, numCPU, interval, publishDelay)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Stop the runner if we receive an interrupt
	go func() {
		sig := <-sigs
		fmt.Fprintf(os.Stderr, "Received signal %s\n", sig)
		r.Stop()
	}()

	err := r.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting runner: %s\n", err)
		os.Exit(1)
	}
	err = r.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running: %s\n", err)
		os.Exit(1)
	}
}

func createNATSConnection(endpoint string) (publisher.PublishingConnection, error) {
	nc, err := nats.Connect(endpoint)
	if err != nil {
		return nil, err
	}
	return nc, nil
}
