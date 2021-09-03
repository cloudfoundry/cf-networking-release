package warrant

import (
	"io"

	"github.com/pivotal-cf-experimental/warrant/internal/network"
)

const schema = "urn:scim:schemas:core:1.0"

var schemas = []string{schema}

// Config contains the primary configuration values for library operation.
type Config struct {
	// Host is a fully qualified url location for the UAA service (ie. https://uaa.example.com).
	Host string

	// SkipVerifySSL is a boolean value indicating whether the HTTP client will validate the SSL
	// certificate of the UAA service should those requests be communicated over HTTPS.
	SkipVerifySSL bool

	// TraceWriter is an io.Writer to which tracing information can be written. This information
	// includes the outgoing request and the incoming responses from UAA.
	TraceWriter io.Writer
}

// Warrant provices access to the users, clients, groups, and tokens services provided by this library.
type Warrant struct {
	config Config

	// Users is a UsersService providing access to the user resource actions.
	Users UsersService

	// Clients is a ClientsService providing access to the client resource actions.
	Clients ClientsService

	// Groups is a GroupsService providing access to the group resource actions.
	Groups GroupsService

	// Tokens is a TokensService providing access to the tokens actions.
	Tokens TokensService
}

// New returns a Warrant initialized with the given Config. The member fields (Users, Clients, Groups,
// and Tokens) have also been initialized with the given Config.
func New(config Config) Warrant {
	return Warrant{
		config:  config,
		Users:   NewUsersService(config),
		Clients: NewClientsService(config),
		Tokens:  NewTokensService(config),
		Groups:  NewGroupsService(config),
	}
}

func newNetworkClient(config Config) network.Client {
	return network.NewClient(network.Config{
		Host:          config.Host,
		SkipVerifySSL: config.SkipVerifySSL,
		TraceWriter:   config.TraceWriter,
	})
}
