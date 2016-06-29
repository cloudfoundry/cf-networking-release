package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type InfoHandler struct {
	Port int
}

func (h *InfoHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	addressStrings := []string{}
	for _, addr := range addrs {
		listenAddr := strings.Split(addr.String(), "/")[0]
		addressStrings = append(addressStrings, listenAddr)
	}

	respBytes, err := json.Marshal(struct {
		ListenAddresses []string
		Port            int
	}{
		ListenAddresses: addressStrings,
		Port:            h.Port,
	})
	if err != nil {
		panic(err)
	}
	resp.Write(respBytes)
	return
}

type ProxyHandler struct{}

func (h *ProxyHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	destination := strings.TrimPrefix(req.URL.Path, "/proxy/")
	destination = "http://" + destination
	getResp, err := http.Get(destination)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %s", err)
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("request failed: %s", err)))
		return
	}
	defer getResp.Body.Close()

	readBytes, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("read body failed: %s", err)))
		return
	}

	resp.Write(readBytes)
}

func launchHandler(port int, proxyHandler http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/proxy/", proxyHandler)
	mux.Handle("/", &InfoHandler{
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

	proxyHandler := &ProxyHandler{}

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
