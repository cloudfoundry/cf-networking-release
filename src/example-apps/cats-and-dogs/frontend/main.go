package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
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
						<div class=".col-md-4.col-md-offset-4">
				  			<div class="form-group">
								<label for="url">Backend URL</label>
								<input type="text" name="url" class="form-control" placeholder="127.0.0.1:8080">
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
	return
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

	return
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

	launchHandler(systemPort, proxyHandler)
}
