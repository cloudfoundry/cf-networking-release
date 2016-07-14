package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type InfoHandler struct {
	Port      int
	UserPorts string
}

func (h *InfoHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	var overlayIP string
	for _, addr := range addrs {
		listenAddr := strings.Split(addr.String(), "/")[0]
		if strings.HasPrefix(listenAddr, "10.255.") {
			overlayIP = listenAddr
		}
	}

	resp.Write([]byte("<html><style>* { font-family: 'arial' }</style><body><p>"))
	resp.Write([]byte("My overlay IP is: "))
	resp.Write([]byte(overlayIP))
	resp.Write([]byte(`</p><p>I'm serving cats on ports `))
	resp.Write([]byte(h.UserPorts))
	resp.Write([]byte(`</p>`))
	resp.Write([]byte("</html></body>"))
	return
}

type CatHandler struct {
	Port int
}

func (h *CatHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Write([]byte("<html><style>* { font-family: 'arial' }</style><body><p>"))
	resp.Write([]byte(`<img src="http://i.imgur.com/1uYroRF.gif" />`))
	resp.Write([]byte(fmt.Sprintf("</p><p>Hello from the backend, port: %d</p>", h.Port)))
	resp.Write([]byte("</html></body>"))
	return
}

func launchCatHandler(port int) {
	mux := http.NewServeMux()
	mux.Handle("/", &CatHandler{
		Port: port,
	})
	httpServer := http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", port),
		Handler: mux,
	}
	httpServer.SetKeepAlivesEnabled(false)
	httpServer.ListenAndServe()
}

func launchInfoHandler(port int, userPorts string) {
	mux := http.NewServeMux()
	mux.Handle("/", &InfoHandler{
		Port:      port,
		UserPorts: userPorts,
	})
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux)
}

func main() {
	systemPortString := os.Getenv("PORT")
	systemPort, err := strconv.Atoi(systemPortString)
	if err != nil {
		log.Fatal("invalid required env var PORT")
	}

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

		go launchCatHandler(userPort)
	}

	launchInfoHandler(systemPort, userPortsString)
}
