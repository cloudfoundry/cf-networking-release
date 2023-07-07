package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"proxy/handlers"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitService struct {
	BindingGUID string `json:"binding_guid"`
	BindingName any    `json:"binding_name"`
	Credentials struct {
		DashboardURL string   `json:"dashboard_url"`
		Hostname     string   `json:"hostname"`
		Hostnames    []string `json:"hostnames"`
		HTTPAPIURI   string   `json:"http_api_uri"`
		HTTPAPIUris  []string `json:"http_api_uris"`
		Password     string   `json:"password"`
		Protocols    struct {
			Amqp struct {
				Host     string   `json:"host"`
				Hosts    []string `json:"hosts"`
				Password string   `json:"password"`
				Port     int      `json:"port"`
				Ssl      bool     `json:"ssl"`
				URI      string   `json:"uri"`
				Uris     []string `json:"uris"`
				Username string   `json:"username"`
				Vhost    string   `json:"vhost"`
			} `json:"amqp"`
		} `json:"protocols"`
		Ssl      bool     `json:"ssl"`
		URI      string   `json:"uri"`
		Uris     []string `json:"uris"`
		Username string   `json:"username"`
		Vhost    string   `json:"vhost"`
	} `json:"credentials"`
	InstanceGUID   string   `json:"instance_guid"`
	InstanceName   string   `json:"instance_name"`
	Label          string   `json:"label"`
	Name           string   `json:"name"`
	Plan           string   `json:"plan"`
	Provider       any      `json:"provider"`
	SyslogDrainURL any      `json:"syslog_drain_url"`
	Tags           []string `json:"tags"`
	VolumeMounts   []any    `json:"volume_mounts"`
}

type Services struct {
	Rabbit []RabbitService `json:"p.rabbitmq"`
}

func main() {
	systemPortString := os.Getenv("PORT")
	port, err := strconv.Atoi(systemPortString)
	if err != nil {
		log.Fatal("invalid required env var PORT")
	}
	stats := &handlers.Stats{Latency: []float64{}}

	servicesList := getServicesList()

	mux := http.NewServeMux()
	mux.Handle("/", &handlers.InfoHandler{Port: port})
	mux.Handle("/dig/", &handlers.DigHandler{})
	mux.Handle("/digudp/", &handlers.DigUDPHandler{})
	mux.Handle("/download/", &handlers.DownloadHandler{})
	mux.Handle("/dumprequest/", &handlers.DumpRequestHandler{})
	mux.Handle("/echosourceip", &handlers.EchoSourceIPHandler{})
	mux.Handle("/ping/", &handlers.PingHandler{})
	mux.Handle("/proxy/", &handlers.ProxyHandler{Stats: stats})
	mux.Handle("/stats", &handlers.StatsHandler{Stats: stats})
	mux.Handle("/timed_dig/", &handlers.TimedDigHandler{})
	mux.Handle("/upload", &handlers.UploadHandler{})
	mux.Handle("/eventuallyfail", &handlers.EventuallyFailHandler{})
	if containsRabbitCreds(servicesList) {
		fmt.Println("🐰 Setting up rabbit")
		ch := setupRabbit(servicesList.Rabbit[0])
		mux.Handle("/spam", &handlers.QueueHandler{Ch: ch})
	}

	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux)
}

func getServicesList() Services {
	vcapServices := []byte(os.Getenv("VCAP_SERVICES"))

	var servicesList Services
	err := json.Unmarshal(vcapServices, &servicesList)
	if err != nil {
		log.Fatal("VCAP_SERVICES failed to unmarshal", err)
	}

	return servicesList
}

func containsRabbitCreds(servicesList Services) bool {
	return servicesList.Rabbit != nil
}

func setupRabbit(rbt RabbitService) *amqp.Channel {

	connectRabbitMQ, err := amqp.Dial(rbt.Credentials.URI)
	if err != nil {
		panic(err)
	}

	channelRabbitMQ, err := connectRabbitMQ.Channel()
	if err != nil {
		panic(err)
	}

	// This is a hack. We are not closing anything. Do not copy this pattern.

	return channelRabbitMQ
}
