package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

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

type FormPage struct {
	Stylesheet  template.HTML
	Cachebuster int
}

var formPageTemplate string = `
<!DOCTYPE html>
<html lang="en">
	<head>
		<title>Frontend</title>
		<meta charset="utf-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
		<meta name="viewport" content="width=device-width, initial-scale=1">
		{{.Stylesheet}}
	</head>
	<body>
		<div class="container">
			<div class="header clearfix navbar navbar-inverse">
				<div class="container">
					<h3>Frontend Sample App</h3>
				</div>
			</div>

			<div class="jumbotron">
				<form action="/proxy/" method="get" class="form-inline">
					<div class="row">
					<h2>HTTP Test</h2>
						<div class=".col-md-4.col-md-offset-4">
				  		<div class="form-group">
								<label for="url">Backend HTTP URL</label>
								<input type="text" name="url" class="form-control" placeholder="127.0.0.1:8080">
							</div>
							<input type="hidden" name="cachebuster" value="{{.Cachebuster}}">
							<button type="submit" class="btn btn-default">Submit</button>
						</div>
  					</div>
				</form>
			</div>

			<div class="jumbotron">
				<form action="/udp-test/" method="get" class="form-inline">
					<div class="row">
						<h2>UDP Test</h2>
						<div class=".col-md-4.col-md-offset-4">
				  		<div class="form-group">
								<label for="url">Backend UDP Server Address</label>
								<input type="text" name="url" class="form-control" placeholder="127.0.0.1:9001">
							</div>
							<br>
				  		<div class="form-group">
								<label for="message">Message</label>
								<input type="text" name="message" class="form-control" placeholder="hello world">
							</div>
							<input type="hidden" name="cachebuster" value="{{.Cachebuster}}">
							<button type="submit" class="btn btn-default">Submit</button>
						</div>
  				</div>
				</form>
			</div>
		</div>
	</body>
</html>
`

type ProxyPage struct {
	Stylesheet template.HTML
	CatBody    template.HTML
}

var proxyPageTemplate string = `
<!DOCTYPE html>
<html lang="en">
	<head>
		<title>Frontend</title>
		<meta charset="utf-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
		<meta name="viewport" content="width=device-width, initial-scale=1">
		{{.Stylesheet}}
	</head>
	<body>
		<div class="container">
			<div class="header clearfix navbar navbar-inverse">
				<div class="container">
					<h3>Frontend Sample App</h3>
				</div>
			</div>

			{{.CatBody}}
		</div>
	</body>
</html>
`

type UDPTestResultPage struct {
	Stylesheet          template.HTML
	TestRequestMessage  string
	TestResponseMessage string
}

var udpTestResultPageTemplate string = `
<!DOCTYPE html>
<html lang="en">
	<head>
		<title>Frontend</title>
		<meta charset="utf-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
		<meta name="viewport" content="width=device-width, initial-scale=1">
		{{.Stylesheet}}
	</head>
	<body>
		<div class="container">
			<div class="header clearfix navbar navbar-inverse">
				<div class="container">
					<h3>Frontend Sample App</h3>
				</div>
			</div>

			<div class="jumbotron">
				<p>You sent the message: <code>{{.TestRequestMessage}}</code> </p>
				<p>Backend UDP server replied: <code>{{.TestResponseMessage}}</code> </p>
			</div>
		</div>
	</body>
</html>
`

type ErrorPage struct {
	Stylesheet template.HTML
	Error      error
}

var errorPageTemplate string = `
<!DOCTYPE html>
<html lang="en">
	<head>
		<title>Frontend</title>
		<meta charset="utf-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
		<meta name="viewport" content="width=device-width, initial-scale=1">
		{{.Stylesheet}}
	</head>
	<body>
		<div class="container">
			<div class="header clearfix navbar navbar-inverse">
				<div class="container">
					<h3>Frontend Sample App</h3>
				</div>
			</div>

			<div class="jumbotron">
				<p><img src="http://i2.kym-cdn.com/photos/images/original/000/234/765/b7e.jpg" /></p>
				<p class="lead">request failed: {{.Error}}</p>
			</div>
		</div>
	</body>
</html>
`

type InfoHandler struct {
	Port int
}

func (h *InfoHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	template := template.Must(template.New("formPage").Parse(formPageTemplate))
	err := template.Execute(resp, FormPage{
		Stylesheet:  stylesheet,
		Cachebuster: rand.Int(),
	})
	if err != nil {
		panic(err)
	}
}

type UDPTestHandler struct{}

func doUDPTest(backendAddress, requestMessage string) (string, error) {
	serverAddr, err := net.ResolveUDPAddr("udp", backendAddress)
	if err != nil {
		return "", fmt.Errorf("resolve udp address: %s", err)
	}

	connection, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return "", fmt.Errorf("dial udp: %s", err)
	}

	err = connection.SetDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return "", fmt.Errorf("set deadline: %s", err)
	}

	defer connection.Close()

	_, err = connection.Write([]byte(requestMessage))
	if err != nil {
		return "", fmt.Errorf("write udp data: %s", err)
	}

	buffer := make([]byte, 1024)
	numBytesReceived, _, err := connection.ReadFromUDP(buffer)
	if err != nil {
		return "", fmt.Errorf("read udp data: %s", err)
	}

	responseMessage := string(buffer[:numBytesReceived])
	return responseMessage, nil
}

func (u *UDPTestHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	queryParams, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		panic(err)
	}
	backendURL := queryParams["url"][0]
	requestMessage := queryParams["message"][0]

	responseMessage, err := doUDPTest(backendURL, requestMessage)
	if err != nil {
		template := template.Must(template.New("formPage").Parse(errorPageTemplate))
		err = template.Execute(resp, ErrorPage{
			Stylesheet: stylesheet,
			Error:      err,
		})
		if err != nil {
			panic(err)
		}

		return
	}

	theTemplate := template.Must(template.New("udpTestPage").Parse(udpTestResultPageTemplate))
	err = theTemplate.Execute(resp, UDPTestResultPage{
		Stylesheet:          stylesheet,
		TestRequestMessage:  requestMessage,
		TestResponseMessage: responseMessage,
	})
	if err != nil {
		panic(err)
	}

}

type ProxyHandler struct{}

func (h *ProxyHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	queryParams, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		panic(err)
	}
	destination := queryParams["url"][0]
	destination = "http://" + destination
	httpClient := http.DefaultClient
	httpClient.Timeout = 5 * time.Second
	getResp, err := httpClient.Get(destination)
	if err != nil {
		template := template.Must(template.New("formPage").Parse(errorPageTemplate))
		err = template.Execute(resp, ErrorPage{
			Stylesheet: stylesheet,
			Error:      err,
		})
		if err != nil {
			panic(err)
		}

		return
	}
	defer getResp.Body.Close()

	readBytes, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("read body failed: %s", err)))
		return
	}

	theTemplate := template.Must(template.New("proxyPage").Parse(proxyPageTemplate))
	catBody := template.HTML(string(readBytes))
	err = theTemplate.Execute(resp, ProxyPage{
		Stylesheet: stylesheet,
		CatBody:    catBody,
	})
	if err != nil {
		panic(err)
	}
}

func launchHandler(port int, proxyHandler http.Handler, udpTestHandler http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/proxy/", proxyHandler)
	mux.Handle("/", &InfoHandler{
		Port: port,
	})
	mux.Handle("/udp-test/", udpTestHandler)
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux)
}

func main() {
	systemPortString := os.Getenv("PORT")
	systemPort, err := strconv.Atoi(systemPortString)
	if err != nil {
		log.Fatal("invalid required env var PORT")
	}

	proxyHandler := &ProxyHandler{}
	udpTestHandler := &UDPTestHandler{}

	launchHandler(systemPort, proxyHandler, udpTestHandler)
}
