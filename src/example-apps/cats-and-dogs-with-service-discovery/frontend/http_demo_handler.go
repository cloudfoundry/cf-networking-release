package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type HttpDemoResultPage struct {
	Stylesheet template.HTML
	CatBody    template.HTML
}

var httpDemoResultPageTemplate string = `
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

type HttpDemoHandler struct{}

func (h *HttpDemoHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
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
		template := template.Must(template.New("errorPageTemplate").Parse(errorPageTemplate))
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

	theTemplate := template.Must(template.New("httpDemoResultPage").Parse(httpDemoResultPageTemplate))
	catBody := template.HTML(string(readBytes))
	err = theTemplate.Execute(resp, HttpDemoResultPage{
		Stylesheet: stylesheet,
		CatBody:    catBody,
	})
	if err != nil {
		panic(err)
	}
}
