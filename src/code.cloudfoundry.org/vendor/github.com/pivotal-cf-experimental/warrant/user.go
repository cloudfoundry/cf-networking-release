package warrant

import (
	"time"

	"github.com/pivotal-cf-experimental/warrant/internal/documents"
)

// User is the representation of a user resource within UAA.
type User struct {
	// ID is the unique identifier for the user.
	ID string

	// ExternalID is an identifier for the user as defined by the client that created it.
	ExternalID string

	// UserName is a human-friendly unique identifier for the user.
	UserName string

	// FormattedName is the full name, including middle names, of the user.
	FormattedName string

	// FamilyName is the family name, or last name, of the user.
	FamilyName string

	// GivenName is the given name, or first name, of the user.
	GivenName string

	// MiddleName is the middle name(s) of the user.
	MiddleName string

	// CreatedAt is a timestamp value indicating when the user was created.
	CreatedAt time.Time

	// UpdatedAt is a timestamp value indicating when the user was last modified.
	UpdatedAt time.Time

	// Version is an integer value indicating which revision this resource represents.
	Version int

	// Emails is a list of email addresses for this user.
	Emails []string

	// Groups is a list of groups to which this user is associated.
	Groups []Group

	// Active is a boolean value indicating the active status of the user.
	Active bool

	// Verified is a boolean value indicating whether this user has been verified.
	Verified bool

	// Origin is a value indicating where this user resource originated.
	Origin string
}

func newUserFromResponse(config Config, response documents.UserResponse) User {
	var emails []string
	for _, email := range response.Emails {
		emails = append(emails, email.Value)
	}

	return User{
		ID:            response.ID,
		ExternalID:    response.ExternalID,
		UserName:      response.UserName,
		FormattedName: response.Name.Formatted,
		FamilyName:    response.Name.FamilyName,
		GivenName:     response.Name.GivenName,
		MiddleName:    response.Name.MiddleName,
		Emails:        emails,
		CreatedAt:     response.Meta.Created,
		UpdatedAt:     response.Meta.LastModified,
		Active:        response.Active,
		Verified:      response.Verified,
		Origin:        response.Origin,
	}
}
