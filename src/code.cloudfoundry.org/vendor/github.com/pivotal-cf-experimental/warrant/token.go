package warrant

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt"
)

// Token is the representation of a token within UAA.
type Token struct {
	// Algorithm is the method used to sign the token.
	Algorithm string

	// KeyID is the ID of the signing key used to sign this token.
	KeyID string

	// ClientID is the value given in the "client_id" field of the token claims.
	// This is the unique identifier of the client to whom this token was granted.
	ClientID string `json:"client_id"`

	// UserID is the value given in the "user_id" field of the token claims.
	// This is the unique identifier for the user.
	UserID string `json:"user_id"`

	// Scopes are the values given in the "scope" field of the token claims.
	// These values indicate the level of access granted by the user to this token.
	Scopes []string `json:"scope"`

	// Issuer is the UAA endpoint that generated the token.
	Issuer string `json:"iss"`

	// Segments contains the raw token segment strings.
	Segments TokenSegments
}

// Verify will use the given signing keys to verify the authenticity of the
// token. Supports RSA and HMAC siging methods.
func (t Token) Verify(signingKeys []SigningKey) error {
	for _, signingKey := range signingKeys {
		if signingKey.KeyId == t.KeyID {
			signingString := strings.Join([]string{t.Segments.Header, t.Segments.Claims}, ".")

			switch t.Algorithm {
			case jwt.SigningMethodRS256.Alg(), jwt.SigningMethodRS384.Alg(), jwt.SigningMethodRS512.Alg():
				method := jwt.GetSigningMethod(t.Algorithm)

				var err error
				key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(signingKey.Value))
				if err != nil {
					block, rest := pem.Decode([]byte(signingKey.Value))
					if len(rest) > 0 {
						return errors.New("public key is not valid PEM encoding")
					}

					key, err = x509.ParsePKCS1PublicKey(block.Bytes)
					if err != nil {
						return err
					}
				}

				return method.Verify(signingString, t.Segments.Signature, key)

			case jwt.SigningMethodHS256.Alg(), jwt.SigningMethodHS384.Alg(), jwt.SigningMethodHS512.Alg():
				method := jwt.GetSigningMethod(t.Algorithm)
				key := []byte(signingKey.Value)
				return method.Verify(signingString, t.Segments.Signature, key)

			default:
				return fmt.Errorf("unsupported token signing method: %s", t.Algorithm)
			}
		}
	}

	return errors.New("token was not signed by a known key")
}

// TokenSegments is the encoded token segments split into their named parts.
type TokenSegments struct {
	// Header is the raw token header segment.
	Header string

	// Claims is the raw token claims segment.
	Claims string

	// Signature is the raw token signature segment.
	Signature string
}
