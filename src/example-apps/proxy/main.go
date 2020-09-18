package main

import (
	"example-apps/proxy/handlers"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	systemPortString := os.Getenv("PORT")
	port, err := strconv.Atoi(systemPortString)
	if err != nil {
		log.Fatal("invalid required env var PORT")
	}
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

	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux)
}
