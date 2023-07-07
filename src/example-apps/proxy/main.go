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
)

var db *sql.DB

type Service struct {
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

type Services map[string][]Service

func main() {
	systemPortString := os.Getenv("PORT")
	port, err := strconv.Atoi(systemPortString)
	if err != nil {
		log.Fatal("invalid required env var PORT")
	}
	stats := &handlers.Stats{Latency: []float64{}}

	vcapServices := []byte(os.Getenv("VCAP_SERVICES"))
	fmt.Println(vcapServices)

	var servicesList Services
	err = json.Unmarshal(vcapServices, &servicesList)

	if err != nil {
		fmt.Println(err)
	}

	dbCreds := servicesList["p.mysql"][0].Credentials

	cfg := mysql.Config{
		User:   dbCreds.Username,
		Passwd: dbCreds.Password,
		Net:    "tcp",
		Addr:   fmt.Sprint(dbCreds.Hostname, ":", dbCreds.Port),
		DBName: "amelia-test",
	}

	// Get a database handle.
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")

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

	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux)
}
