package copilot

import (
	"context"
	"crypto/tls"
	"fmt"

	"code.cloudfoundry.org/bosh-dns-adapter/copilot/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials"
)

//go:generate counterfeiter -o fakes/vip_resolver_copilot_client.go --fake-name VIPResolverCopilotClient api VIPResolverCopilotClient

type Client struct {
	VIPResolverCopilotClient api.VIPResolverCopilotClient
	conn                     *grpc.ClientConn
}

func NewConnectedClient(serverAddr string, dialOpts ...DialOption) (*Client, error) {
	opts := &options{}

	for _, dialOpt := range dialOpts {
		dialOpt(opts)
	}

	grpcOpts := []grpc.DialOption{
		grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingConfig": [{"%s":{}}]}`, roundrobin.Name)),
	}

	if opts.transportCredentials != nil {
		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(opts.transportCredentials))
	}

	if opts.withInsecure {
		grpcOpts = append(grpcOpts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(serverAddr, grpcOpts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:                     conn,
		VIPResolverCopilotClient: api.NewVIPResolverCopilotClient(conn),
	}, err
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) IP(name string) (string, error) {
	//deadline?
	ctx := context.Background()
	req := api.GetVIPByNameRequest{Fqdn: name}

	response, err := c.VIPResolverCopilotClient.GetVIPByName(ctx, &req)
	return response.GetIp(), err
}

type DialOption func(*options)

func WithInsecure() DialOption {
	return func(o *options) {
		o.withInsecure = true
	}
}

func WithTLSConfig(config *tls.Config) DialOption {
	return func(o *options) {
		o.transportCredentials = credentials.NewTLS(config)
	}
}

type options struct {
	transportCredentials credentials.TransportCredentials
	withInsecure         bool
}
