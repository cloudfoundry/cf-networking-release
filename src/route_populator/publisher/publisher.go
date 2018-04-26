package publisher

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type ConnectionCreator func(endpoint string) (PublishingConnection, error)

//go:generate counterfeiter -o fakes/fake_publishing_connection.go . PublishingConnection
type PublishingConnection interface {
	Publish(subj string, data []byte) error
	Close()
}

type Job struct {
	PublishingEndpoint string

	BackendHost string
	BackendPort int

	AppDomain  string
	AppName    string
	StartRange int
	EndRange   int
}

func (j Job) validate() error {
	validationError := func(err string) error {
		return fmt.Errorf("Invalid job properties: %s", err)
	}
	if j.PublishingEndpoint == "" {
		return validationError(`Missing "AppDomain"`)
	}

	if j.BackendHost == "" {
		return validationError(`Missing "BackendHost"`)
	}

	if j.BackendPort == 0 {
		return validationError(`Missing "BackendPort"`)
	}

	if j.AppDomain == "" {
		return validationError(`Missing "AppDomain"`)
	}

	if j.AppName == "" {
		return validationError(`Missing "AppName"`)
	}

	if j.EndRange <= j.StartRange {
		return validationError(`Invalid route "StartRange" and "EndRange"`)
	}
	return nil
}

type RouteData struct {
	Host string   `json:"host"`
	Port int      `json:"port"`
	URIs []string `json:"uris"`
}

type Publisher struct {
	job          Job
	publishDelay time.Duration
	conn         PublishingConnection

	data [][]byte
}

func NewPublisher(job Job, publishDelay time.Duration) *Publisher {
	return &Publisher{
		job:          job,
		publishDelay: publishDelay,
	}
}

func (p *Publisher) Initialize(c ConnectionCreator) error {
	err := p.job.validate()
	if err != nil {
		return err
	}

	conn, err := c(p.job.PublishingEndpoint)
	p.conn = conn
	if err != nil {
		return err
	}

	routeData := RouteData{
		Host: p.job.BackendHost,
		Port: p.job.BackendPort,
	}

	for i := p.job.StartRange; i < p.job.EndRange; i += 1 {
		routeData.URIs = []string{fmt.Sprintf("%s-%d.%s", p.job.AppName, i, p.job.AppDomain)}
		marshaledData, err := json.Marshal(routeData)
		if err != nil {
			return err
		}
		p.data = append(p.data, marshaledData)
	}
	return nil
}

func (p *Publisher) PublishRouteRegistrations() error {
	start := time.Now()
	for i := 0; i < (p.job.EndRange - p.job.StartRange); i += 1 {
		err := p.conn.Publish("router.register", p.data[i])
		if err != nil {
			return err
		}
		err = p.conn.Publish("service-discovery.register", p.data[i])
		if err != nil {
			return err
		}
		time.Sleep(p.publishDelay)
	}
	ttp := time.Since(start)
	log.Printf("Routes published in %f seconds: %d - %d, e.g. %s\n", ttp.Seconds(), p.job.StartRange, p.job.EndRange, string(p.data[0]))
	return nil
}

func (p *Publisher) Finish() {
	p.conn.Close()
}
