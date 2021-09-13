package warrant

// Token is the representation of a token within UAA.
type Token struct {
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
}
