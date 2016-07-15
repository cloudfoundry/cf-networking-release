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
<style>
* {
	font-family: 'arial';
	font-size: 30px;
}
h2 {
	font-size: 32px;
}
</style>
`)

type FormPage struct {
	Stylesheet  template.HTML
	Cachebuster int
}

var formPageTemplate string = `
<html>
	<head>{{.Stylesheet}}</head>
	<body>
		<h2>Frontend</h2>
		<form action="/proxy/" method="get">
			<label>Backend URL:<input type="text" name="url"/></label>
			<input type="hidden" name="cachebuster" value="{{.Cachebuster}}">
			<input type="submit">
		</form>
	</body>
</html>
`

type ErrorPage struct {
	Stylesheet template.HTML
	Error      error
}

var errorPageTemplate string = `
<html>
	<head>{{.Stylesheet}}</head>
	<body>
		<p><img src="http://i2.kym-cdn.com/photos/images/original/000/234/765/b7e.jpg" /></p>
		<p>request failed: {{.Error}}</p>
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
	httpClient.Timeout = 30 * time.Second
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

	launchHandler(systemPort, proxyHandler)
}
