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
<!-- Latest compiled and minified CSS -->
<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap.min.css" integrity="sha384-1q8mTJOASx8j1Au+a5WDVnPi2lkFfwwEAa8hDDdjZlpLegxhjVME1fgjWPGmkzs7" crossorigin="anonymous">

<!-- Optional theme -->
<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap-theme.min.css" integrity="sha384-fLW2N01lMqjakBkx3l/M9EahuwpSfeNvV63J5ezn3uZzapT0u7EYsXMjQV+0En5r" crossorigin="anonymous">

<!-- Latest compiled and minified JavaScript -->
<script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/js/bootstrap.min.js" integrity="sha384-0mSbJDEHialfmuBBQP6A4Qrprq5OVfW37PRR3j5ELqxss1yVqOtnepnHVP9aJ7xS" crossorigin="anonymous"></script>
<style>
.jumbotron {
	text-align: center;
}
.header h3 {
	color: white;
}
</style>
`)

type PublicPage struct {
	Stylesheet template.HTML
	OverlayIP  string
	UserPorts  string
}

var publicPageTemplate string = `
<!DOCTYPE html>
<html lang="en">
  <head>
	<title>Backend</title>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
	{{.Stylesheet}}
	</head>
	<body>
		<div class="container">
			<div class="header clearfix navbar navbar-inverse">
				<div class="container">
					<h3>Backend Sample App</h3>
				</div>
			</div>
			<div class="jumbotron">
				<h1>My overlay IP is: {{.OverlayIP}}</h1>
				<p class="lead">I'm serving cats on TCP ports {{.UserPorts}}</p>
			</div>
		</div>
	</body>
</html>
`

type CatPage struct {
	Stylesheet template.HTML
	Port       int
}

var catPageTemplate string = `
<!DOCTYPE html>
<html lang="en">
	<head>
		<title>Backend</title>
		<meta charset="utf-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
		<meta name="viewport" content="width=device-width, initial-scale=1">
		{{.Stylesheet}}
	</head>
	<body>
		<div class="row">
			<div class="col-xs-8 col-xs-offset-2">
				<div class="header clearfix navbar navbar-inverse">
					<div class="container">
						<h3>Backend Sample App</h3>
					</div>
				</div>
				<div class="jumbotron">
					<p class="lead">Hello from the backend, here is a picture of a cat:</p>
					<p><img src="http://i.imgur.com/1uYroRF.gif" /></p>
				</div>
			</div>
		</div>
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
