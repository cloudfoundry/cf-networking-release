package documents

// TokenResponse represents the JSON transport data structure
// for a request that returns a token value.
type TokenResponse struct {
	// AccessToken is the token string used to authenticate
	// with UAA-based services.
	AccessToken string `json:"access_token"`

	// TokenType describes the type of token returned.
	// This value is always "Bearer".
	TokenType string `json:"token_type"`

	// ExpiresIn is the number of seconds until this token
	// expires.
	ExpiresIn int `json:"expires_in"`

	// Scope is a comma separated list of permission values
	// for this token.
	Scope string `json:"scope"`

	// JTI is the unique identifier for this JWT token.
	JTI string `json:"jti"`

	// Issuer is the URL to the issuer of the token.
	Issuer string `json:"iss"`
}

// TokenKeyResponse represents the JSON transport data structure
// for a response from UAA containing the token signing key.
type TokenKeyResponse struct {
	// Alg is the algorithm that this key is used for.
	Alg string `json:"alg"`

	// Value is a string representation of the key.
	Value string `json:"value"`

	// Kty identifies the cryptographic algorithm family used with the key.
	Kty string `json:"kty"`

	// Use identifies the intended use of the public key. Use is employed
	// to indicate whether a public key is used for encrypting data or
	// verifying the signature on data.
	// Values defined by the JWT specification are:
	// - sig (signature)
	// - enc (encryption)
	Use string `json:"use"`

	// N is the public/private modulus for the key.
	N string `json:"n"`

	// E is the public exponent for the key.
	E string `json:"e"`
}
