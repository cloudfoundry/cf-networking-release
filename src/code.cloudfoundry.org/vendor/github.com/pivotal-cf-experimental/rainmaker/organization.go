package rainmaker

import (
	"net/url"
	"time"

	"github.com/pivotal-cf-experimental/rainmaker/internal/documents"
)

type Organization struct {
	config                   Config
	GUID                     string
	Name                     string
	URL                      string
	BillingEnabled           bool
	Status                   string
	QuotaDefinitionGUID      string
	QuotaDefinitionURL       string
	SpacesURL                string
	DomainsURL               string
	PrivateDomainsURL        string
	UsersURL                 string
	ManagersURL              string
	BillingManagersURL       string
	AuditorsURL              string
	AppEventsURL             string
	SpaceQuotaDefinitionsURL string
	CreatedAt                time.Time
	UpdatedAt                time.Time
	Users                    UsersList
	BillingManagers          UsersList
	Auditors                 UsersList
	Managers                 UsersList
}

func NewOrganization(config Config, guid string) Organization {
	return Organization{
		config:          config,
		GUID:            guid,
		Users:           NewUsersList(config, newRequestPlan("/v2/organizations/"+guid+"/users", url.Values{})),
		BillingManagers: NewUsersList(config, newRequestPlan("/v2/organizations/"+guid+"/billing_managers", url.Values{})),
		Auditors:        NewUsersList(config, newRequestPlan("/v2/organizations/"+guid+"/auditors", url.Values{})),
		Managers:        NewUsersList(config, newRequestPlan("/v2/organizations/"+guid+"/managers", url.Values{})),
	}
}

func newOrganizationFromResponse(config Config, response documents.OrganizationResponse) Organization {
	if response.Metadata.CreatedAt == nil {
		response.Metadata.CreatedAt = &time.Time{}
	}

	if response.Metadata.UpdatedAt == nil {
		response.Metadata.UpdatedAt = &time.Time{}
	}

	organization := NewOrganization(config, response.Metadata.GUID)
	organization.URL = response.Metadata.URL
	organization.CreatedAt = *response.Metadata.CreatedAt
	organization.UpdatedAt = *response.Metadata.UpdatedAt
	organization.Name = response.Entity.Name
	organization.BillingEnabled = response.Entity.BillingEnabled
	organization.Status = response.Entity.Status
	organization.QuotaDefinitionGUID = response.Entity.QuotaDefinitionGUID
	organization.QuotaDefinitionURL = response.Entity.QuotaDefinitionURL
	organization.SpacesURL = response.Entity.SpacesURL
	organization.DomainsURL = response.Entity.DomainsURL
	organization.PrivateDomainsURL = response.Entity.PrivateDomainsURL
	organization.UsersURL = response.Entity.UsersURL
	organization.ManagersURL = response.Entity.ManagersURL
	organization.BillingManagersURL = response.Entity.BillingManagersURL
	organization.AuditorsURL = response.Entity.AuditorsURL
	organization.AppEventsURL = response.Entity.AppEventsURL
	organization.SpaceQuotaDefinitionsURL = response.Entity.SpaceQuotaDefinitionsURL

	return organization
}
