package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type InfoHandler struct {
	Port int
}

func (h *InfoHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Write([]byte("<html><style>* { font-family: 'arial' }</style><body><p>"))
	resp.Write([]byte(fmt.Sprintf(`
		</p>
		<h2>Frontend</h2>
		<form action="/proxy/" method="get">
		<label>Backend URL:<input type="text" name="url"/></label><input type="submit">
		<input type="hidden" name="cachebuster" value="%d">
		</form>`, rand.Int())))
	resp.Write([]byte("</html></body>"))
	return
}

type ProxyHandler struct{}

func (h *ProxyHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	log.Printf("rawquery: %+v\n", req.URL.RawQuery)
	queryParams, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		panic(err)
	}
	destination := queryParams["url"][0]
	destination = "http://" + destination
	httpClient := http.DefaultClient
	httpClient.Timeout = 30 * time.Second
	getResp, err := httpClient.Get(destination)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %s", err)
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte("<html><style>* { font-family: 'arial' }</style><body><p>"))
		resp.Write([]byte(`<img src="http://i2.kym-cdn.com/photos/images/original/000/234/765/b7e.jpg" /></p><p>`))
		resp.Write([]byte(fmt.Sprintf("request failed: %s", err)))
		resp.Write([]byte("</p></html></body>"))
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
