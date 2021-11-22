package warrant

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/pivotal-cf-experimental/warrant/internal/documents"
	"github.com/pivotal-cf-experimental/warrant/internal/network"
)

// TokensService provides access to common token actions. Using this service,
// you can decode a token and fetch the signing key to validate a token.
type TokensService struct {
	config Config
}

// SigningKey is the representation of the key used to validate a token.
type SigningKey struct {
	// id for the signing key
	KeyId string

	// Algorithm indicates the kind of key used to sign tokens.
	// Keys can be either symmetric or asymmetric.
	Algorithm string

	// Value is a string representation of the key. In the case of a symmetric key,
	// this is the shared secret value. for asymmetric keys, this is the public key
	// of the keypair.
	Value string
}

// NewTokensService returns a TokensService initialized with the given Config.
func NewTokensService(config Config) TokensService {
	return TokensService{
		config: config,
	}
}

// Decode returns a decoded token value. The returned value represents the
// token's claims section.
func (ts TokensService) Decode(token string) (Token, error) {
	segments := strings.Split(token, ".")
	if len(segments) != 3 {
		return Token{}, InvalidTokenError{fmt.Errorf("invalid number of segments in token (%d/3)", len(segments))}
	}

	headerSegment, err := jwt.DecodeSegment(segments[0])
	if err != nil {
		return Token{}, InvalidTokenError{fmt.Errorf("header cannot be decoded: %s", err)}
	}

	var header struct {
		Alg   string `json:"alg"`
		KeyID string `json:"kid"`
	}
	err = json.Unmarshal(headerSegment, &header)
	if err != nil {
		return Token{}, InvalidTokenError{fmt.Errorf("header cannot be parsed: %s", err)}
	}

	claims, err := jwt.DecodeSegment(segments[1])
	if err != nil {
		return Token{}, InvalidTokenError{fmt.Errorf("claims cannot be decoded: %s", err)}
	}

	t := Token{
		Algorithm: header.Alg,
		KeyID:     header.KeyID,
		Segments: TokenSegments{
			Header:    segments[0],
			Claims:    segments[1],
			Signature: segments[2],
		},
	}
	err = json.Unmarshal(claims, &t)
	if err != nil {
		return Token{}, InvalidTokenError{fmt.Errorf("token cannot be parsed: %s", err)}
	}

	return t, nil
}

// GetSigningKey makes a request to UAA to retrieve the SigningKey used to
// generate valid tokens.
func (ts TokensService) GetSigningKey() (SigningKey, error) {
	resp, err := newNetworkClient(ts.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  "/token_key",
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return SigningKey{}, translateError(err)
	}

	var response documents.TokenKeyResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return SigningKey{}, MalformedResponseError{err}
	}

	key := SigningKey{
		KeyId:     response.Kid,
		Algorithm: response.Alg,
		Value:     response.Value,
	}

	return key, nil
}

// GetSigningKeys makes a request to UAA to retrieve the SigningKeys used to
// generate valid tokens.
func (ts TokensService) GetSigningKeys() ([]SigningKey, error) {
	resp, err := newNetworkClient(ts.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  "/token_keys",
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return []SigningKey{}, translateError(err)
	}

	var response documents.TokenKeysResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return []SigningKey{}, MalformedResponseError{err}
	}

	signingKeys := make([]SigningKey, 0, len(response.Keys))

	for _, key := range response.Keys {
		signingKeys = append(signingKeys, SigningKey{
			KeyId:     key.Kid,
			Algorithm: key.Alg,
			Value:     key.Value,
		})
	}

	return signingKeys, nil
}
