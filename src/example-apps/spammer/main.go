package main

import (
	"log"
	"net/http"

	"spammer/api"
)

func main() {
	http.HandleFunc(api.SpamPath, api.SpamHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("An error occured during serving: %s", err)
	}
}
