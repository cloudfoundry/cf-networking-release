package main

import (
	"encoding/json"
	"example-apps/tick/a8"
	"fmt"
	"log"
	"net/http"
	"os"

	"code.cloudfoundry.org/localip"

	"github.com/ryanmoran/viron"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

type Environment struct {
	VCAPApplication struct {
		ApplicationName string `json:"application_name"`
		InstanceIndex   int    `json:"instance_index"`
	} `env:"VCAP_APPLICATION" env-required:"true"`

	Port            string `env:"PORT" env-required:"true"`
	RegistryBaseURL string `env:"REGISTRY_BASE_URL"  env-required:"true"`
}

func main() {
	if err := mainWithError(); err != nil {
		log.Printf("%s", err)
		os.Exit(1)
	}
}

func mainWithError() error {
	var env Environment
	err := viron.Parse(&env)
	if err != nil {
		return fmt.Errorf("unable to parse environment: %s", err)
	}

	localIP, err := localip.LocalIP()
	if err != nil {
		return fmt.Errorf("unable to discover local ip: %s", err)
	}

	a8Client := &a8.Client{
		BaseURL:            env.RegistryBaseURL,
		HttpClient:         http.DefaultClient,
		LocalServerAddress: fmt.Sprintf("%s:%s", localIP, env.Port),
		ServiceName:        env.VCAPApplication.ApplicationName,
		TTLSeconds:         10,
	}
	err = a8Client.Register()
	if err != nil {
		return fmt.Errorf("a8 register: %s", err)
	}

	infoHandler := &InfoHandler{
		InfoData: env.VCAPApplication,
	}
	server := http_server.New("0.0.0.0:"+env.Port, infoHandler)

	members := grouper.Members{
		{"http_server", server},
	}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))

	err = <-monitor.Wait()
	if err != nil {
		return fmt.Errorf("ifrit monitor: %s", err)
	}

	return nil
}

type InfoHandler struct {
	InfoData interface{}
}

func (h *InfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(h.InfoData)
}
