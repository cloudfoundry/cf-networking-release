package main

import (
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"time"
)

type UDPDemoResultPage struct {
	Stylesheet          template.HTML
	TestRequestMessage  string
	TestResponseMessage string
}

var udpDemoResultPageTemplate string = `
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

type UDPDemoHandler struct{}

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

func (u *UDPDemoHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
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

	theTemplate := template.Must(template.New("udpTestPage").Parse(udpDemoResultPageTemplate))
	err = theTemplate.Execute(resp, UDPDemoResultPage{
		Stylesheet:          stylesheet,
		TestRequestMessage:  requestMessage,
		TestResponseMessage: responseMessage,
	})
	if err != nil {
		panic(err)
	}

}
