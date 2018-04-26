package mbus

import (
	"encoding/json"
	"time"

	"sync"

	"net/url"

	"fmt"

	"os"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/nats-io/nats"
	"github.com/pkg/errors"
)

const (
	registerMessagesReceived = "registerMessagesReceived"
)

type ServiceDiscoveryStartMessage struct {
	Id                               string `json:"id"`
	Host                             string `json:"host"`
	MinimumRegisterIntervalInSeconds int    `json:"minimumRegisterIntervalInSeconds"`
	PruneThresholdInSeconds          int    `json:"pruneThresholdInSeconds"`
}

type SubscriberOpts struct {
	ID                               string
	MinimumRegisterIntervalInSeconds int
	PruneThresholdInSeconds          int
	AcceptTLS                        bool
}

type RegistryMessage struct {
	IP                string   `json:"host"`
	InfraNames        []string `json:"uris"`
	EndpointUpdatedAt int64    `json:"endpoint_updated_at_ns"`
}

//go:generate counterfeiter -o fakes/address_table.go --fake-name AddressTable . AddressTable
type AddressTable interface {
	Add(infraNames []string, ip string)
	Remove(infraNames []string, ip string)
	PausePruning()
	ResumePruning()
	SetWarm()
}

type Subscriber struct {
	natsConnProvider NatsConnProvider
	subOpts          SubscriberOpts
	warmingDuration  time.Duration
	table            AddressTable
	logger           lager.Logger
	recorder         routeMessageRecorder
	localIP          string
	natsClient       NatsConn
	once             sync.Once
	metricsSender    metricsSender
	clock            clock.Clock
}

//go:generate counterfeiter -o fakes/nats_conn.go --fake-name NatsConn . NatsConn
type NatsConn interface {
	PublishMsg(m *nats.Msg) error
	Close()
	Flush() error
	ConnectedUrl() string
	Subscribe(string, nats.MsgHandler) (*nats.Subscription, error)
}

//go:generate counterfeiter -o fakes/nats_conn_provider.go --fake-name NatsConnProvider . NatsConnProvider
type NatsConnProvider interface {
	Connection(opts ...nats.Option) (NatsConn, error)
}

//go:generate counterfeiter -o fakes/metrics_sender.go --fake-name MetricsSender . metricsSender
type metricsSender interface {
	IncrementCounter(string)
}

//go:generate counterfeiter -o fakes/route_message_recorder.go --fake-name RouteMessageRecorder . routeMessageRecorder
type routeMessageRecorder interface {
	RecordMessageTransitTime(time int64)
}

func NewSubscriber(
	natsConnProvider NatsConnProvider,
	subOpts SubscriberOpts,
	warmingDuration time.Duration,
	table AddressTable,
	localIP string,
	recorder routeMessageRecorder,
	logger lager.Logger,
	metricsSender metricsSender,
	clock clock.Clock,
) *Subscriber {
	return &Subscriber{
		natsConnProvider: natsConnProvider,
		subOpts:          subOpts,
		warmingDuration:  warmingDuration,
		table:            table,
		recorder:         recorder,
		logger:           logger,
		localIP:          localIP,
		metricsSender:    metricsSender,
		clock:            clock,
	}
}

func (s *Subscriber) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	err := s.RunOnce()
	if err != nil {
		return err
	}

	close(ready)

	for {
		select {
		case <-signals:
			s.Close()
			return nil
		}
	}
}

func (s *Subscriber) RunOnce() error {
	var err error
	s.once.Do(func() {
		var natsClient NatsConn
		natsClient, err = s.natsConnProvider.Connection(
			nats.ReconnectHandler(nats.ConnHandler(func(conn *nats.Conn) {
				{
					url, err := url.Parse(conn.ConnectedUrl())
					if err == nil {
						s.logger.Info(
							"ReconnectHandler reconnected to nats server",
							lager.Data{"nats_host": url.Scheme + "://" + url.Host}, //don't leak creds
						)
					}
				}

				s.table.ResumePruning()

				s.sendStartMessage()
			})),
			nats.DisconnectHandler(nats.ConnHandler(func(conn *nats.Conn) {
				s.logger.Info(
					"DisconnectHandler disconnected from nats server",
					lager.Data{"last_error": conn.LastError()},
				)

				s.table.PausePruning()
			})),
			nats.ClosedHandler(nats.ConnHandler(func(conn *nats.Conn) {
				s.logger.Info(
					"ClosedHandler unexpected close of nats connection",
					lager.Data{"last_error": conn.LastError()},
				)
			})),
			nats.MaxReconnects(-1),
		)

		if err != nil {
			err = errors.Wrap(err, "unable to create nats connection")
			return
		}

		s.natsClient = natsClient

		{
			url, err := url.Parse(natsClient.ConnectedUrl())
			if err == nil {
				s.logger.Info(
					"Connected to NATS server",
					lager.Data{"nats_host": url.Scheme + "://" + url.Host},
				)
			}
		}

		err = s.sendStartMessage()
		if err != nil {
			return
		}

		err = s.setupGreetMsgHandler()
		if err != nil {
			return
		}

		err = s.setupAddressMessageHandler()
		if err != nil {
			return
		}

		go func() {
			<-s.clock.After(s.warmingDuration)
			s.table.SetWarm()
		}()
	})

	if err != nil {
		s.Close()
	}

	return err
}

func (s *Subscriber) sendStartMessage() error {
	msg := &nats.Msg{
		Subject: "service-discovery.start",
		Data:    s.subscriptionOptionsJSON(),
	}

	err := s.natsClient.PublishMsg(msg)
	if err != nil {
		return errors.Wrap(err, "unable to publish a start message")
	}

	s.logger.Info("service-discovery.start-message-published", lager.Data(map[string]interface{}{
		"subscription_options": string(s.subscriptionOptionsJSON()),
	}))

	return nil
}

func (s *Subscriber) Close() {
	if s.natsClient != nil {
		s.natsClient.Close()
	}
}

func (s *Subscriber) setupGreetMsgHandler() error {
	discoveryMessageJson := s.subscriptionOptionsJSON()

	_, err := s.natsClient.Subscribe("service-discovery.greet", nats.MsgHandler(func(greetMsg *nats.Msg) {
		err := s.natsClient.PublishMsg(&nats.Msg{
			Subject: greetMsg.Reply,
			Data:    discoveryMessageJson,
		})

		if err != nil {
			s.logger.Error("GreetMsgHandler unable to publish response to greet messages", err)
		}

		s.logger.Info("service-discovery.greet-response-published")
	}))

	if err != nil {
		s.logger.Error("setupGreetMsgHandler unable to subscribe to greet messages", err)
		return err
	}

	err = s.natsClient.Flush()
	if err != nil {
		s.logger.Error("setupGreetMsgHandler unable to flush subscribe greet message", err)
		return err
	}

	return nil
}

func (s *Subscriber) setupAddressMessageHandler() error {
	_, err := s.natsClient.Subscribe("service-discovery.register", nats.MsgHandler(func(msg *nats.Msg) {
		registryMessage := &RegistryMessage{}
		err := json.Unmarshal(msg.Data, registryMessage)
		if err != nil || registryMessage.IP == "" || len(registryMessage.InfraNames) == 0 {
			s.logger.Info("AddressMessageHandler received a malformed register message", lager.Data(map[string]interface{}{
				"msgJson": string(msg.Data),
			}))
			return
		}

		s.recorder.RecordMessageTransitTime(registryMessage.EndpointUpdatedAt)
		s.metricsSender.IncrementCounter(registerMessagesReceived)

		s.logger.Debug("AddressMessageHandler register msg received", lager.Data(map[string]interface{}{
			"msgJson": string(msg.Data),
		}))
		s.table.Add(registryMessage.InfraNames, registryMessage.IP)
	}))

	if err != nil {
		s.logger.Error("setupAddressMessageHandler unable to subscribe to service-discovery.register", err)
		return err
	}

	_, err = s.natsClient.Subscribe("service-discovery.unregister", nats.MsgHandler(func(msg *nats.Msg) {
		registryMessage := &RegistryMessage{}
		err := json.Unmarshal(msg.Data, registryMessage)
		if err != nil || len(registryMessage.InfraNames) == 0 {
			s.logger.Info("AddressMessageHandler received a malformed unregister message", lager.Data(map[string]interface{}{
				"msgJson": string(msg.Data),
			}))
			return
		}
		s.logger.Debug("AddressMessageHandler unregister msg received", lager.Data(map[string]interface{}{
			"msgJson": string(msg.Data),
		}))
		s.table.Remove(registryMessage.InfraNames, registryMessage.IP)
	}))

	if err != nil {
		s.logger.Error("setupAddressMessageHandler unable to subscribe to service-discovery.unregister", err)
		return err
	}

	return nil
}

func (s *Subscriber) subscriptionOptionsJSON() []byte {
	discoveryMessageJson, err := json.Marshal(ServiceDiscoveryStartMessage{
		Id:   s.subOpts.ID,
		Host: s.localIP,
		MinimumRegisterIntervalInSeconds: s.subOpts.MinimumRegisterIntervalInSeconds,
		PruneThresholdInSeconds:          s.subOpts.PruneThresholdInSeconds,
	})

	if err != nil {
		panic(fmt.Sprintf("Unable to marshal subsription options: %v", err))
	}

	return discoveryMessageJson
}
