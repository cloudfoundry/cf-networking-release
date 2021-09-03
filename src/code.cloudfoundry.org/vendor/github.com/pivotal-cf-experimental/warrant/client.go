package warrant

import (
	"time"

	"github.com/pivotal-cf-experimental/warrant/internal/documents"
)

// Client is the representation of a client resource within UAA.
type Client struct {
	// ID is the unique identifier for the client resource.
	ID string

	Name string

	// Scope contains a list of scope values describing the level of permissions for a
	// user token requested by this client.
	Scope []string

	// Authorities is a list of scope values describing the level of permissions granted
	// to this client in a token requested with the "client_credentials" grant type.
	Authorities []string

	// ResourceIDs is a white list of resource identifiers to be included in the decoded
	// tokens granted to this client. The UAA does not store any data here (it should be
	// "none" for all clients), but instead creates a list of resource identifiers
	// dynamically from the scope values when a token is granted.
	ResourceIDs []string

	// AuthorizedGrantTypes is a list of OAuth2 grant types, as defined in the spec.
	// Valid fields are:
	//   - client_credentials
	//   - password
	//   - implicit
	//   - refresh_token
	//   - authorization_code
	AuthorizedGrantTypes []string

	// AccessTokenValidity is the number of seconds before a token granted to this client
	// will expire.
	AccessTokenValidity time.Duration

	// RedirectURI is the location address to redirect the resource owner's user-agent
	// back to after completing its interaction with the resource owner.
	RedirectURI []string

	// Autoapprove is a list of scopes to automatically approve when making an implicit
	// grant for a user token.
	Autoapprove []string
}

func newClientFromDocument(document documents.ClientResponse) Client {
	return Client{
		ID:                   document.ClientID,
		Name:                 document.Name,
		Scope:                sort(document.Scope),
		ResourceIDs:          sort(document.ResourceIDs),
		Authorities:          sort(document.Authorities),
		AuthorizedGrantTypes: sort(document.AuthorizedGrantTypes),
		Autoapprove:          sort(document.Autoapprove),
		AccessTokenValidity:  time.Duration(document.AccessTokenValidity) * time.Second,
		RedirectURI:          document.RedirectURI,
	}
}

func (c Client) toDocument(secret string) documents.CreateUpdateClientRequest {
	client := documents.CreateUpdateClientRequest{
		ClientID:             c.ID,
		ClientSecret:         secret,
		Name:                 c.Name,
		AccessTokenValidity:  int(c.AccessTokenValidity.Seconds()),
		Scope:                make([]string, 0),
		ResourceIDs:          make([]string, 0),
		Authorities:          make([]string, 0),
		AuthorizedGrantTypes: make([]string, 0),
		RedirectURI:          make([]string, 0),
		Autoapprove:          make([]string, 0),
	}
	client.Scope = append(client.Scope, c.Scope...)
	client.ResourceIDs = append(client.ResourceIDs, c.ResourceIDs...)
	client.Authorities = append(client.Authorities, c.Authorities...)
	client.AuthorizedGrantTypes = append(client.AuthorizedGrantTypes, c.AuthorizedGrantTypes...)
	client.RedirectURI = append(client.RedirectURI, c.RedirectURI...)
	client.Autoapprove = append(client.Autoapprove, c.Autoapprove...)

	return client
}
