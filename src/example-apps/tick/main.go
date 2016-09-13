package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

func main() {

	port := os.Getenv("PORT")
	if port == "" {
		log.Printf("missing required env var PORT")
		os.Exit(1)
	}

	server := http_server.New("0.0.0.0:"+port, http.HandlerFunc(index))

	members := grouper.Members{
		{"http_server", server},
	}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))

	err := <-monitor.Wait()
	if err != nil {
		os.Exit(1)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello")
}
