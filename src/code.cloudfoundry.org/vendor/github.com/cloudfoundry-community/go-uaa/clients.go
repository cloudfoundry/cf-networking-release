package uaa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ClientsEndpoint is the path to the clients resource.
const ClientsEndpoint string = "/oauth/clients"

// paginatedClientList is the response from the API for a single page of clients.
type paginatedClientList struct {
	Page
	Resources []Client `json:"resources"`
	Schemas   []string `json:"schemas"`
}

// Client is a UAA client
// http://docs.cloudfoundry.org/api/uaa/version/4.19.0/index.html#clients.
type Client struct {
	ClientID             string          `json:"client_id,omitempty" generator:"id"`
	AuthorizedGrantTypes []string        `json:"authorized_grant_types,omitempty"`
	RedirectURI          []string        `json:"redirect_uri,omitempty"`
	Scope                []string        `json:"scope,omitempty"`
	ResourceIDs          []string        `json:"resource_ids,omitempty"`
	Authorities          []string        `json:"authorities,omitempty"`
	AutoApproveRaw       interface{}     `json:"autoapprove,omitempty"`
	AccessTokenValidity  int64           `json:"access_token_validity,omitempty"`
	RefreshTokenValidity int64           `json:"refresh_token_validity,omitempty"`
	AllowedProviders     []string        `json:"allowedproviders,omitempty"`
	DisplayName          string          `json:"name,omitempty"`
	TokenSalt            string          `json:"token_salt,omitempty"`
	CreatedWith          string          `json:"createdwith,omitempty"`
	ApprovalsDeleted     bool            `json:"approvals_deleted,omitempty"`
	RequiredUserGroups   []string        `json:"required_user_groups,omitempty"`
	ClientSecret         string          `json:"client_secret,omitempty"`
	LastModified         int64           `json:"lastModified,omitempty"`
	AllowPublic          bool            `json:"allowpublic,omitempty"`
	JwksURI              string          `json:"jwks_uri,omitempty"`
	Jwks                 json.RawMessage `json:"jwks,omitempty"`
}

// UnmarshalJSON decodes a Client, tolerating UAA responses that encode
// allowpublic, approvals_deleted, lastModified, allowedproviders, and
// required_user_groups with a different JSON type than usual (e.g. a JSON
// string instead of a boolean, or a single string instead of an array). UAA
// stores these fields in a loosely-typed map and serializes back whatever
// type was originally stored there, so the same field can come back as a
// bool on one client and a string on another. AllowPublic, ApprovalsDeleted,
// LastModified, AllowedProviders, and RequiredUserGroups keep their normal
// public field types; only decoding is more lenient.
func (c *Client) UnmarshalJSON(data []byte) error {
	type alias Client
	aux := struct {
		AllowPublic        interface{} `json:"allowpublic,omitempty"`
		ApprovalsDeleted   interface{} `json:"approvals_deleted,omitempty"`
		LastModified       interface{} `json:"lastModified,omitempty"`
		AllowedProviders   interface{} `json:"allowedproviders,omitempty"`
		RequiredUserGroups interface{} `json:"required_user_groups,omitempty"`
		*alias
	}{
		alias: (*alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	c.AllowPublic = clientRawToBool(aux.AllowPublic)
	c.ApprovalsDeleted = clientRawToBool(aux.ApprovalsDeleted)
	c.LastModified = clientRawToInt64(aux.LastModified)
	c.AllowedProviders = clientRawToStringSlice(aux.AllowedProviders)
	c.RequiredUserGroups = clientRawToStringSlice(aux.RequiredUserGroups)

	return nil
}

func clientRawToBool(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, err := strconv.ParseBool(t)
		if err != nil {
			return false
		}
		return b
	}
	return false
}

func clientRawToInt64(v interface{}) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case string:
		i, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return 0
		}
		return i
	}
	return 0
}

// clientRawToStringSlice handles the fact that a JSON array decoded into an
// interface{} becomes []interface{}, not []string.
func clientRawToStringSlice(v interface{}) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		return []string{t}
	case []interface{}:
		result := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return []string{}
}

// Identifier returns the field used to uniquely identify a Client.
func (c Client) Identifier() string {
	return c.ClientID
}

// AutoApprove tolerates UAA responses that encode autoapprove as a JSON
// boolean, string, or array (a JSON array decoded into AutoApproveRaw
// becomes []interface{}, not []string).
func (c Client) AutoApprove() []string {
	switch t := c.AutoApproveRaw.(type) {
	case bool:
		return []string{strconv.FormatBool(t)}
	case string:
		return []string{t}
	case []string:
		return t
	case []interface{}:
		result := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return []string{}
}

// GrantType is a type of oauth2 grant.
type GrantType string

// Valid GrantType values.
const (
	REFRESHTOKEN      = GrantType("refresh_token")
	AUTHCODE          = GrantType("authorization_code")
	IMPLICIT          = GrantType("implicit")
	PASSWORD          = GrantType("password")
	CLIENTCREDENTIALS = GrantType("client_credentials")
)

func errorMissingValueForGrantType(value string, grantType GrantType) error {
	return fmt.Errorf("%v must be specified for %v grant type", value, grantType)
}

func errorMissingValue(value string) error {
	return fmt.Errorf("%v must be specified in the client definition", value)
}

func requireRedirectURIForGrantType(c *Client, grantType GrantType) error {
	if contains(c.AuthorizedGrantTypes, string(grantType)) {
		if len(c.RedirectURI) == 0 {
			return errorMissingValueForGrantType("redirect_uri", grantType)
		}
	}
	return nil
}

func requireClientSecretForGrantType(c *Client, grantType GrantType) error {
	if contains(c.AuthorizedGrantTypes, string(grantType)) {
		if c.ClientSecret == "" {
			return errorMissingValueForGrantType("client_secret", grantType)
		}
	}
	return nil
}

func knownGrantTypesStr() string {
	grantTypeStrings := []string{}
	knownGrantTypes := []GrantType{AUTHCODE, IMPLICIT, PASSWORD, CLIENTCREDENTIALS}
	for _, grant := range knownGrantTypes {
		grantTypeStrings = append(grantTypeStrings, string(grant))
	}

	return "[" + strings.Join(grantTypeStrings, ", ") + "]"
}

// Validate returns nil if the client is valid, or an error if it is invalid.
func (c *Client) Validate() error {
	if len(c.AuthorizedGrantTypes) == 0 {
		return fmt.Errorf("grant type must be one of %v", knownGrantTypesStr())
	}

	if c.ClientID == "" {
		return errorMissingValue("client_id")
	}

	if err := requireRedirectURIForGrantType(c, AUTHCODE); err != nil {
		return err
	}
	if err := requireClientSecretForGrantType(c, AUTHCODE); err != nil {
		return err
	}

	if err := requireClientSecretForGrantType(c, CLIENTCREDENTIALS); err != nil {
		return err
	}

	if err := requireRedirectURIForGrantType(c, IMPLICIT); err != nil {
		return err
	}

	return nil
}

type changeSecretBody struct {
	ClientID     string `json:"clientId,omitempty"`
	ClientSecret string `json:"secret,omitempty"`
}

// ChangeClientSecret updates the secret with the given value for the client
// with the given id
// http://docs.cloudfoundry.org/api/uaa/version/4.14.0/index.html#change-secret.
func (a *API) ChangeClientSecret(id string, newSecret string) error {
	u := urlWithPath(*a.TargetURL, fmt.Sprintf("%s/%s/secret", ClientsEndpoint, id))
	change := &changeSecretBody{ClientID: id, ClientSecret: newSecret}
	j, err := json.Marshal(change)
	if err != nil {
		return err
	}
	err = a.doJSON(http.MethodPut, &u, bytes.NewBuffer([]byte(j)), nil, true)
	if err != nil {
		return err
	}
	return nil
}

// ChangeClientJWTMode is the operation mode for ChangeClientJWT.
type ChangeClientJWTMode string

const (
	ChangeClientJWTModeAdd    = ChangeClientJWTMode("ADD")
	ChangeClientJWTModeUpdate = ChangeClientJWTMode("UPDATE")
	ChangeClientJWTModeDelete = ChangeClientJWTMode("DELETE")
)

// ClientJWTChangeRequest is the request body for the PUT /oauth/clients/{id}/clientjwt endpoint.
type ClientJWTChangeRequest struct {
	ClientID   string              `json:"client_id,omitempty"`
	ChangeMode ChangeClientJWTMode `json:"changeMode,omitempty"`
	JwksURI    string              `json:"jwks_uri,omitempty"`
	Jwks       json.RawMessage     `json:"jwks,omitempty"`
	Kid        string              `json:"kid,omitempty"`
	Issuer     string              `json:"iss,omitempty"`
	Subject    string              `json:"sub,omitempty"`
	Audience   string              `json:"aud,omitempty"`
}

// ChangeClientJWT configures the JWT trust for the client with the given id.
// Use jwks_uri or jwks to specify public keys; use iss/sub/aud for federation JWT trust.
// changeMode controls whether the key is ADDed, UPDATEd, or DELETEd (kid required for DELETE).
func (a *API) ChangeClientJWT(req ClientJWTChangeRequest) error {
	if req.ClientID == "" {
		return errorMissingValue("client_id")
	}
	if req.ChangeMode == ChangeClientJWTModeDelete && req.Kid == "" {
		return fmt.Errorf("kid must be specified when changeMode is %v", ChangeClientJWTModeDelete)
	}
	u := urlWithPath(*a.TargetURL, fmt.Sprintf("%s/%s/clientjwt", ClientsEndpoint, req.ClientID))
	j, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return a.doJSON(http.MethodPut, &u, bytes.NewBuffer(j), nil, true)
}
