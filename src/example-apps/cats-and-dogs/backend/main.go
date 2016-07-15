package main

import (
	"fmt"
	"html/template"
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

var stylesheet template.HTML = template.HTML(`
<style>
* {
	font-family: 'arial';
	font-size: 30px;
}
</style>
`)

type PublicPage struct {
	Stylesheet template.HTML
	OverlayIP  string
	UserPorts  string
}

var publicPageTemplate string = `
<html>
	<head>{{.Stylesheet}}</head>
	<body>
		<p>My overlay IP is: {{.OverlayIP}}</p>
		<p>I'm serving cats on ports {{.UserPorts}}</p>
	</body>
</html>
`

type CatPage struct {
	Stylesheet template.HTML
	Port       int
}

var catPageTemplate string = `
<html>
	<head>{{.Stylesheet}}</head>
	<body>
		<p><img src="http://i.imgur.com/1uYroRF.gif" /></p>
		<p>Hello from the backend, port: {{.Port}}</p>
	</body>
</html>
`

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

	template := template.Must(template.New("publicPage").Parse(publicPageTemplate))
	err = template.Execute(resp, PublicPage{
		Stylesheet: stylesheet,
		OverlayIP:  overlayIP,
		UserPorts:  h.UserPorts,
	})
	if err != nil {
		panic(err)
	}
	return
}

type CatHandler struct {
	Port int
}

func (h *CatHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	template := template.Must(template.New("catPage").Parse(catPageTemplate))
	err := template.Execute(resp, CatPage{
		Stylesheet: stylesheet,
		Port:       h.Port,
	})
	if err != nil {
		panic(err)
	}
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
