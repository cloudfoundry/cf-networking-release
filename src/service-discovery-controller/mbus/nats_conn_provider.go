package mbus

import (
	"crypto/tls"

	"github.com/nats-io/go-nats"
)

type NatsConnWithUrlProvider struct {
	Url       string
	TLSConfig *tls.Config
}

func (p *NatsConnWithUrlProvider) Connection(opts ...nats.Option) (NatsConn, error) {
	if p.TLSConfig != nil {
		opts = append(opts, nats.Secure(p.TLSConfig))
	}
	return nats.Connect(p.Url, opts...)
}
