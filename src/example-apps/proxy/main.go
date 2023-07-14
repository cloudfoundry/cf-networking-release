package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"proxy/handlers"
	"strconv"

	"github.com/go-sql-driver/mysql"
	amqp "github.com/rabbitmq/amqp091-go"
)

var db *sql.DB

type MysqlService struct {
	BindingGUID string `json:"binding_guid,omitempty"`
	BindingName any    `json:"binding_name,omitempty"`
	Credentials struct {
		Hostname string `json:"hostname,omitempty"`
		JdbcURL  string `json:"jdbcUrl,omitempty"`
		Name     string `json:"name,omitempty"`
		Password string `json:"password,omitempty"`
		Port     int    `json:"port,omitempty"`
		TLS      struct {
			Cert struct {
				Ca string `json:"ca,omitempty"`
			} `json:"cert,omitempty"`
		} `json:"tls,omitempty"`
		URI      string `json:"uri,omitempty"`
		Username string `json:"username,omitempty"`
	} `json:"credentials,omitempty"`
	InstanceGUID   string   `json:"instance_guid,omitempty"`
	InstanceName   string   `json:"instance_name,omitempty"`
	Label          string   `json:"label,omitempty"`
	Name           string   `json:"name,omitempty"`
	Plan           string   `json:"plan,omitempty"`
	Provider       any      `json:"provider,omitempty"`
	SyslogDrainURL any      `json:"syslog_drain_url,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	VolumeMounts   []any    `json:"volume_mounts,omitempty"`
}
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
	Mysql  []MysqlService  `json:"p.mysql"`
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
	setupRabbit(servicesList.Rabbit[0])

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
	if containsDBCreds(servicesList) {
		db := setUpDB(servicesList)
		mux.Handle("/todos", &handlers.TodosHandler{Db: db})
	}
	if containsRabbitCreds(servicesList) {
		fmt.Println("🐰")
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

func containsDBCreds(servicesList Services) bool {
	return servicesList.Mysql[0].BindingGUID != ""
}

func containsRabbitCreds(servicesList Services) bool {
	return servicesList.Rabbit[0].BindingGUID != ""
}

func setupRabbit(rbt RabbitService) {
	connectRabbitMQ, err := amqp.Dial(rbt.Credentials.URI)
	if err != nil {
		panic(err)
	}
	defer connectRabbitMQ.Close()

	// Let's start by opening a channel to our RabbitMQ
	// instance over the connection we have already
	// established.
	channelRabbitMQ, err := connectRabbitMQ.Channel()
	if err != nil {
		panic(err)
	}
	defer channelRabbitMQ.Close()

	_, err = channelRabbitMQ.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		panic(err)
	}
}

func setUpDB(servicesList Services) *sql.DB {
	return nil

	dbCreds := servicesList.Mysql[0].Credentials

	cfg := mysql.Config{
		User:   dbCreds.Username,
		Passwd: dbCreds.Password,
		Net:    "tcp",
		Addr:   fmt.Sprint(dbCreds.Hostname, ":", dbCreds.Port),
		// DBName:               "amelia-test",
		AllowNativePasswords: true,
	}

	// Get a database handle.
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")

	dbName := "ameliatest"
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS " + dbName)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("🐈")

	_, err = db.Exec("USE " + dbName)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("🦇")
	dbTableName := "todos"
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS " + dbTableName + " ( done bool, note varchar(32) )")
	if err != nil {
		log.Fatal(err)
	}

	return db
}
