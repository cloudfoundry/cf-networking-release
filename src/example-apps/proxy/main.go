package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"proxy/handlers"
	"strconv"
)

const succeedAfterDefault = 5
const failAfterDefault = 5
const flapIntervalDefault = 5

func main() {
	port := getEnvVar("PORT", 0, true)
	failAfterCount := getEnvVar("EVENTUALLY_FAIL_AFTER_COUNT", failAfterDefault, false)
	succeedAfterCount := getEnvVar("EVENTUALLY_SUCCEED_AFTER_COUNT", succeedAfterDefault, false)
	flapInterval := getEnvVar("FLAP_INTERVAL", flapIntervalDefault, false)
	stats := &handlers.Stats{Latency: []float64{}}

	mux := http.NewServeMux()
	mux.Handle("/", &handlers.InfoHandler{Port: port})
	mux.Handle("/dig/", &handlers.DigHandler{})
	mux.Handle("/digudp/", &handlers.DigUDPHandler{})
	mux.Handle("/download/", &handlers.DownloadHandler{})
	mux.Handle("/dumprequest/", &handlers.DumpRequestHandler{})
	mux.Handle("/echosourceip", &handlers.EchoSourceIPHandler{})
	mux.Handle("/ping/", &handlers.PingHandler{})
	mux.Handle("/proxy/", &handlers.ProxyHandler{Stats: stats})
	mux.Handle("/stats", &handlers.StatsHandler{Stats: stats})
	mux.Handle("/timed_dig/", &handlers.TimedDigHandler{})
	mux.Handle("/upload", &handlers.UploadHandler{})
	mux.Handle("/eventuallyfail", &handlers.EventuallyFailHandler{FailAfterCount: failAfterCount})
	mux.Handle("/eventuallysucceed", &handlers.EventuallySucceedHandler{SucceedAfterCount: succeedAfterCount})
	mux.Handle("/flap", &handlers.FlapHandler{FlapInterval: flapInterval})
	mux.Handle("/signal/", &handlers.SignalHandler{})

	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux)
}

func getEnvVar(key string, defaultValue int, failIfDNE bool) int {
	var result int
	var err error

	v, ok := os.LookupEnv(key)
	if !ok && failIfDNE {
		log.Fatalf("invalid required env var %s", key)
	} else if !ok {
		return defaultValue
	}

	result, err = strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return result
}
