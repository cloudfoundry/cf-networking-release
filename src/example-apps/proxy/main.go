package main

import (
	"example-apps/proxy/handlers"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func launchHandler(port int, proxyHandler http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/proxy/", proxyHandler)
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

	proxyHandler := &handlers.ProxyHandler{}

	userPortsString := os.Getenv("USER_PORTS")
	userPorts := strings.Split(userPortsString, ",")
	for _, userPortString := range userPorts {
		if strings.TrimSpace(userPortString) == "" {
			continue
		}
		userPort, err := strconv.Atoi(userPortString)
		if err != nil {
			log.Fatal("invalid user port " + userPortString)
		}

		go launchHandler(userPort, proxyHandler)
	}

	launchHandler(systemPort, proxyHandler)
}
