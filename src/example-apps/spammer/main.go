package main

import (
	"log"
	"net/http"
	"time"

	"spammer/api"
)

func main() {
	http.HandleFunc(api.SpamPath, api.SpamHandler)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           nil,
		ReadHeaderTimeout: 5 * time.Second,
	}
	err := server.ListenAndServe()
	log.Fatalf("An error occured during serving: %s", err)
}
