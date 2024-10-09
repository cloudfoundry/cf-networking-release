package main

import (
	"log"
	"net/http"

	"spammer/api"
)

func main() {
	http.HandleFunc(api.SpamPath, api.SpamHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: nil,
	}
	err := server.ListenAndServe()
	log.Fatalf("An error occured during serving: %s", err)
}
