package documents

import "time"

type SpaceResponse struct {
	Metadata struct {
		GUID      string     `json:"guid"`
		URL       string     `json:"url"`
		CreatedAt *time.Time `json:"created_at"`
		UpdatedAt *time.Time `json:"updated_at"`
	} `json:"metadata"`
	Entity struct {
		Name                     string `json:"name"`
		OrganizationGUID         string `json:"organization_guid"`
		SpaceQuotaDefinitionGUID string `json:"space_quota_definition_guid"`
		OrganizationURL          string `json:"organization_url"`
		DevelopersURL            string `json:"developers_url"`
		ManagersURL              string `json:"managers_url"`
		AuditorsURL              string `json:"auditors_url"`
		AppsURL                  string `json:"apps_url"`
		RoutesURL                string `json:"routes_url"`
		DomainsURL               string `json:"domains_url"`
		ServiceInstancesURL      string `json:"service_instances_url"`
		AppEventsURL             string `json:"app_events_url"`
		EventsURL                string `json:"events_url"`
		SecurityGroupsURL        string `json:"security_groups_url"`
	} `json:"entity"`
}
