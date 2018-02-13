package main

import (
	"bytes"
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
				  <p class="lead">You reached me on port {{.Port}}</p>
				</div>
			</div>
		</div>
	</body>
</html>
`

func (h *InfoHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	log.Printf("InfoHandler: request received from %s", req.RemoteAddr)
	return
}

type CatHandler struct {
	Port int
}

func (h *CatHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	log.Printf("CatHandler: request received from %s", req.RemoteAddr)
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

func generateReply(requestMessage []byte) []byte {
	return bytes.ToUpper(requestMessage)
}

func handleUDPConnection(connection *net.UDPConn) error {
	buffer := make([]byte, 1024)

	numBytesReceived, clientAddress, err := connection.ReadFromUDP(buffer)
	if err != nil {
		return fmt.Errorf("reading udp packet: %s", err)
	}

	requestMessage := buffer[:numBytesReceived]
	log.Printf("UDP client: %s sent message %s", clientAddress, string(requestMessage))

	replyMessage := generateReply(requestMessage)

	_, err = connection.WriteToUDP(replyMessage, clientAddress)
	log.Printf("replied with: %s", string(replyMessage))
	return err
}

func launchUDPServer(port int) {
	udpAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		panic(err)
	}

	connection, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}

	defer connection.Close()

	for {
		err := handleUDPConnection(connection)
		if err != nil {
			log.Panicf("handle UDP connection: %s", err)
		}
	}
}

func launchInfoHandler(port int) {
	mux := http.NewServeMux()
	mux.Handle("/", &InfoHandler{})
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux)
}

func main() {
	systemPortString := os.Getenv("PORT")
	systemPort, err := strconv.Atoi(systemPortString)
	log.SetOutput(os.Stdout)
	if err != nil {
		log.Fatal("invalid required env var PORT")
	}

	userPorts, err := extractPortNumbers("CATS_PORTS")
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, userPort := range userPorts {
		go launchCatHandler(userPort)
	}

	udpPorts, err := extractPortNumbers("UDP_PORTS")
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, udpPort := range udpPorts {
		go launchUDPServer(udpPort)
	}

	launchInfoHandler(systemPort)
}

func extractPortNumbers(envVarName string) ([]int, error) {
	portStrings := strings.Split(os.Getenv(envVarName), ",")
	portNumbers := []int{}
	for _, portString := range portStrings {
		if strings.TrimSpace(portString) == "" {
			continue
		}
		portNumber, err := strconv.Atoi(portString)
		if err != nil {
			return nil, fmt.Errorf("invalid port %s", portString)
		}

		portNumbers = append(portNumbers, portNumber)
	}
	return portNumbers, nil
}
