package warrant

import "github.com/pivotal-cf-experimental/warrant/internal/documents"

// Member is the representation of a group member resource within UAA.
// This is probably just a user.
type Member struct {
	// The alias of the identity provider that authenticated
	// this user. "uaa" is an internal UAA user.
	Origin string `json:"origin"`

	// Type is either "USER" or "GROUP".
	Type string `json:"type"`

	// Value is the globally-unique ID of the member entity,
	// either a user ID or another group ID.
	Value string `json:"value"`
}

func newMemberFromResponse(config Config, response documents.MemberResponse) Member {
	return Member{
		Type:   response.Type,
		Value:  response.Value,
		Origin: response.Origin,
	}
}
