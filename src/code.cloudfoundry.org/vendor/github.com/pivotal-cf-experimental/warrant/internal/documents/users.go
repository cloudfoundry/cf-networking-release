package documents

// CreateUserRequest represents the JSON transport data structure
// for a request to create a User.
type CreateUserRequest struct {
	// UserName is the unique identifier for the user resource.
	// This identifier is used by the user to authenticate with
	// the UAA service.
	UserName string `json:"userName"`

	// Name is the components of the real user's name. This field
	// contains several representation of the user's name.
	Name UserName `json:"name"`

	// Emails is a list of email addresses for the user.
	Emails []Email `json:"emails"`
}

// UpdateUserRequest represents the JSON transport data structure
// for a request to update an existing User.
type UpdateUserRequest struct {
	// Schemas is the list of schemas for this API request.
	Schemas []string `json:"schemas"`

	// ID is the unique identifier for this SCIM resource within
	// the UAA service.
	ID string `json:"id"`

	// UserName is the unique identifier for the user resource.
	// This identifier is used by the user to authenticate with
	// the UAA service.
	UserName string `json:"userName"`

	// ExternalID is an identifier for the user as specified by
	// the creator of this resource.
	ExternalID string `json:"externalId"`

	// Name is the components of the real user's name. This field
	// contains several representation of the user's name.
	Name UserName `json:"name"`

	// Emails is a list of email addresses for the user.
	Emails []Email `json:"emails"`

	// Meta is the set of metadata for this resource.
	Meta Meta `json:"meta"`
}

// UserResponse represents the JSON transport data structure
// for a response from UAA containing a user resource.
type UserResponse struct {
	// Schemas is the list of schemas for this API request.
	Schemas []string `json:"schemas"`

	// ID is the unique identifier for this SCIM resource within
	// the UAA service.
	ID string `json:"id"`

	// ExternalID is an identifier for the user as specified by
	// the creator of this resource.
	ExternalID string `json:"externalId"`

	// UserName is the unique identifier for the user resource.
	// This identifier is used by the user to authenticate with
	// the UAA service.
	UserName string `json:"userName"`

	// Name is the components of the real user's name. This field
	// contains several representation of the user's name.
	Name UserName `json:"name"`

	// Emails is a list of email addresses for the user.
	Emails []Email `json:"emails"`

	// Meta is the set of metadata for this resource.
	Meta Meta `json:"meta"`

	// Groups is a list of group resources that the user belongs to.
	Groups []GroupAssociation `json:"groups"`

	// Active is the value indicating the activation status of the user.
	Active bool `json:"active"`

	// Verified is the value indicating whether the user resource has
	// been verified through email.
	Verified bool `json:"verified"`

	// Origin is the name of the UAA provider that the user resource
	// exists within.
	Origin string `json:"origin"`
}

// UserListResponse represents the JSON transport data structure
// for a response from UAA containing a list of user resources.
type UserListResponse struct {
	// Schemas is the list of schemas for this API request.
	Schemas []string `json:"schemas"`

	// Resources is a list of user resources matching the
	// request query.
	Resources []UserResponse `json:"resources"`

	// StartIndex indicates the start of this page of results.
	StartIndex int `json:"startIndex"`

	// ItemsPerPage indicates the number of resources to return
	// in any given request.
	ItemsPerPage int `json:"itemsPerPage"`

	// TotalResults indicates the total number of resources
	// matching the request query.
	TotalResults int `json:"totalResults"`
}

// UserName represents the JSON transport data structure
// for the UserName of a user resource.
type UserName struct {
	// Formatted is the full name of a user, including all
	// middle names, titles, and suffixes as appropriate,
	// formatted for display.
	Formatted string `json:"formatted"`

	// FamilyName is the family name of the user, or "last name"
	// in most Western languages.
	FamilyName string `json:"familyName"`

	// GivenName is the given name of the user, or "first name"
	// in most Western languages.
	GivenName string `json:"givenName"`

	// MiddleName is the middle name(s) of the user.
	MiddleName string `json:"middleName"`
}

// Email represents the JSON transport data structure
// for an email belonging to a user.
type Email struct {
	// Value is the email address represented as a string.
	Value string `json:"value"`
}
