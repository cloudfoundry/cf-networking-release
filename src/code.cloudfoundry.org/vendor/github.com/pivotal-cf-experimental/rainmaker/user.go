package rainmaker

import (
	"time"

	"github.com/pivotal-cf-experimental/rainmaker/internal/documents"
)

type User struct {
	config                         Config
	GUID                           string
	URL                            string
	CreatedAt                      time.Time
	UpdatedAt                      time.Time
	Admin                          bool
	Active                         bool
	DefaultSpaceGUID               string
	SpacesURL                      string
	OrganizationsURL               string
	ManagedOrganizationsURL        string
	BillingManagedOrganizationsURL string
	AuditedOrganizationsURL        string
	ManagedSpacesURL               string
	AuditedSpacesURL               string
}

func NewUser(config Config, guid string) User {
	return User{
		config: config,
		GUID:   guid,
	}
}

func newUserFromResponse(config Config, response documents.UserResponse) User {
	if response.Metadata.CreatedAt == nil {
		response.Metadata.CreatedAt = &time.Time{}
	}

	if response.Metadata.UpdatedAt == nil {
		response.Metadata.UpdatedAt = &time.Time{}
	}

	user := NewUser(config, response.Metadata.GUID)
	user.URL = response.Metadata.URL
	user.CreatedAt = *response.Metadata.CreatedAt
	user.UpdatedAt = *response.Metadata.UpdatedAt
	user.Admin = response.Entity.Admin
	user.Active = response.Entity.Active
	user.DefaultSpaceGUID = response.Entity.DefaultSpaceGUID
	user.SpacesURL = response.Entity.SpacesURL
	user.OrganizationsURL = response.Entity.OrganizationsURL
	user.ManagedOrganizationsURL = response.Entity.ManagedOrganizationsURL
	user.BillingManagedOrganizationsURL = response.Entity.BillingManagedOrganizationsURL
	user.AuditedOrganizationsURL = response.Entity.AuditedOrganizationsURL
	user.ManagedSpacesURL = response.Entity.ManagedSpacesURL
	user.AuditedSpacesURL = response.Entity.AuditedSpacesURL

	return user
}
