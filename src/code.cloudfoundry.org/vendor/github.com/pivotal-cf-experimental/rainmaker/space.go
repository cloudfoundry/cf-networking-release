package rainmaker

import (
	"net/url"
	"time"

	"github.com/pivotal-cf-experimental/rainmaker/internal/documents"
)

type Space struct {
	config                   Config
	GUID                     string
	URL                      string
	CreatedAt                time.Time
	UpdatedAt                time.Time
	Name                     string
	OrganizationGUID         string `json:"organization_guid"` // TODO: why is this here?
	SpaceQuotaDefinitionGUID string
	OrganizationURL          string
	DevelopersURL            string
	ManagersURL              string
	AuditorsURL              string
	AppsURL                  string
	RoutesURL                string
	DomainsURL               string
	ServiceInstancesURL      string
	AppEventsURL             string
	EventsURL                string
	SecurityGroupsURL        string
	Developers               UsersList
}

func NewSpace(config Config, guid string) Space {
	return Space{
		config:     config,
		GUID:       guid,
		Developers: NewUsersList(config, newRequestPlan("/v2/spaces/"+guid+"/developers", url.Values{})),
	}
}

func newSpaceFromResponse(config Config, response documents.SpaceResponse) Space {
	space := NewSpace(config, response.Metadata.GUID)
	if response.Metadata.CreatedAt == nil {
		response.Metadata.CreatedAt = &time.Time{}
	}

	if response.Metadata.UpdatedAt == nil {
		response.Metadata.UpdatedAt = &time.Time{}
	}

	space.URL = response.Metadata.URL
	space.CreatedAt = *response.Metadata.CreatedAt
	space.UpdatedAt = *response.Metadata.UpdatedAt
	space.Name = response.Entity.Name
	space.OrganizationGUID = response.Entity.OrganizationGUID
	space.SpaceQuotaDefinitionGUID = response.Entity.SpaceQuotaDefinitionGUID
	space.OrganizationURL = response.Entity.OrganizationURL
	space.DevelopersURL = response.Entity.DevelopersURL
	space.ManagersURL = response.Entity.ManagersURL
	space.AuditorsURL = response.Entity.AuditorsURL
	space.AppsURL = response.Entity.AppsURL
	space.RoutesURL = response.Entity.RoutesURL
	space.DomainsURL = response.Entity.DomainsURL
	space.ServiceInstancesURL = response.Entity.ServiceInstancesURL
	space.AppEventsURL = response.Entity.AppEventsURL
	space.EventsURL = response.Entity.EventsURL
	space.SecurityGroupsURL = response.Entity.SecurityGroupsURL

	return space
}
