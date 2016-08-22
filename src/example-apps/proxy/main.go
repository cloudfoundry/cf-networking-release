package main

import (
	"example-apps/proxy/handlers"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

func launchHandler(port int, proxyHandler, statsHandler http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/proxy/", proxyHandler)
	mux.Handle("/stats", statsHandler)
	mux.Handle("/", &handlers.InfoHandler{
		Port: port,
	})
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux)
}

func main() {
	systemPortString := os.Getenv("PORT")
	systemPort, err := strconv.Atoi(systemPortString)
	if err != nil {
		log.Fatal("invalid required env var PORT")
	}

	stats := &handlers.Stats{
		Latency: []float64{},
	}
	proxyHandler := &handlers.ProxyHandler{
		Stats: stats,
	}
	statsHandler := &handlers.StatsHandler{
		Stats: stats,
	}

	launchHandler(systemPort, proxyHandler, statsHandler)
}
