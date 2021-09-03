package network

import (
	"encoding/base64"
	"fmt"
)

type authorization interface {
	Authorization() string
}

// NewTokenAuthorization returns a TokenAuthorization initialized
// with the given token value.
func NewTokenAuthorization(token string) TokenAuthorization {
	return TokenAuthorization(token)
}

// TokenAuthorization is an authorization object capable of
// providing a Bearer Token authorization header for a
// request to UAA.
type TokenAuthorization string

// Authorization returns a string that can be used as the value of
// an Authorization HTTP header.
func (a TokenAuthorization) Authorization() string {
	return fmt.Sprintf("Bearer %s", a)
}

// NewBasicAuthorization returns a BasicAuthorization initialized
// with the given username and password.
func NewBasicAuthorization(username, password string) BasicAuthorization {
	return BasicAuthorization{
		username: username,
		password: password,
	}
}

// BasicAuthorization is an authorization object capable of
// providing a HTTP Basic authorization header for a request
// to UAA.
type BasicAuthorization struct {
	username string
	password string
}

// Authorization returns a string that can be used as the value of
// an Authorization HTTP header.
func (b BasicAuthorization) Authorization() string {
	auth := b.username + ":" + b.password
	return fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(auth)))
}
