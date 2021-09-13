package documents

// CreateUpdateClientRequest represents the JSON transport data structure
// for a request to create or update a Client.
type CreateUpdateClientRequest struct {
	// ClientID is the unique identifier specifying the client.
	ClientID string `json:"client_id"`

	// ClientSecret is the secret value used to fetch a token
	// for the client.
	ClientSecret string `json:"client_secret"`

	// Name is the human-friendly name given to a client resource.
	Name string `json:"name"`

	// Scope is a list of permission values to apply to user tokens that
	// are granted to the client.
	Scope []string `json:"scope"`

	// ResourceIDs is a list of audiences for the client. This field
	// is always ["none"].
	ResourceIDs []string `json:"resource_ids"`

	// Authorities is a list of permission values applied when the client
	// fetches their own token.
	Authorities []string `json:"authorities"`

	// AuthorizedGrantTypes is a list of grant types applied to the client.
	AuthorizedGrantTypes []string `json:"authorized_grant_types"`

	// AccessTokenValidity is the number of seconds before a token granted
	// to this client will expire.
	AccessTokenValidity int `json:"access_token_validity"`

	// RedirectURI is the location address to redirect the resource owner's user-agent
	// back to after completing its interaction with the resource owner.
	RedirectURI []string `json:"redirect_uri"`

	// Autoapprove is a list of scopes used to auto-approve a request
	// to fetch a user token.
	Autoapprove []string `json:"autoapprove"`
}

// ClientResponse represents the JSON transport data structure for
// a response containing a Client resource.
type ClientResponse struct {
	// ClientID is the unique identifier specifying the client.
	ClientID string `json:"client_id"`

	// Name is the human-friendly name given to a client resource.
	Name string `json:"name"`

	// Scope is a list of permission values to apply to user tokens that
	// are granted to the client.
	Scope []string `json:"scope"`

	// ResourceIDs is a list of audiences for the client. This field
	// is always ["none"].
	ResourceIDs []string `json:"resource_ids"`

	// Authorities is a list of permission values applied when the client
	// fetches their own token.
	Authorities []string `json:"authorities"`

	// AuthorizedGrantTypes is a list of grant types applied to the client.
	AuthorizedGrantTypes []string `json:"authorized_grant_types"`

	// AccessTokenValidity is the number of seconds before a token granted
	// to this client will expire.
	AccessTokenValidity int `json:"access_token_validity"`

	// RedirectURI is the location address to redirect the resource owner's user-agent
	// back to after completing its interaction with the resource owner.
	RedirectURI []string `json:"redirect_uri"`

	// Autoapprove is a list of scopes used to auto-approve a request
	// to fetch a user token.
	Autoapprove []string `json:"autoapprove"`
}

type ClientListResponse struct {
	// Schemas is the list of schemas for this API request.
	Schemas []string `json:"schemas"`

	// Resources is a list of client resources.
	Resources []ClientResponse `json:"resources"`

	// StartIndex is the index number to start at when returning
	// the list of resources.
	StartIndex int `json:"startIndex"`

	// ItemsPerPage is the number of items to return in the
	// list of resources.
	ItemsPerPage int `json:"itemsPerPage"`

	// TotalResults is the total number of resources that match
	// the list query.
	TotalResults int `json:"totalResults"`
}
