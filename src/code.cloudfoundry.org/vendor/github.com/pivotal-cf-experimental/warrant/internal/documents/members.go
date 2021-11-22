package documents

type CreateMemberRequest struct {
	// The alias of the identity provider that authenticated
	// this user. "uaa" is an internal UAA user.
	Origin string `json:"origin"`

	// Type is either "USER" or "GROUP".
	Type string `json:"type"`

	// Value is the globally-unique ID of the member entity,
	// either a user ID or another group ID.
	Value string `json:"value"`
}

type MemberResponse struct {
	// The alias of the identity provider that authenticated
	// this user. "uaa" is an internal UAA user.
	Origin string `json:"origin"`

	// Type is either "USER" or "GROUP".
	Type string `json:"type"`

	// Value is the globally-unique ID of the member entity,
	// either a user ID or another group ID.
	Value string `json:"value"`
}
