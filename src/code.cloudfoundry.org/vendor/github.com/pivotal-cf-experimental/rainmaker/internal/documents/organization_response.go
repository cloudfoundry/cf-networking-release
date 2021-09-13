package documents

import "time"

type OrganizationResponse struct {
	Metadata struct {
		GUID      string     `json:"guid"`
		URL       string     `json:"url"`
		CreatedAt *time.Time `json:"created_at"`
		UpdatedAt *time.Time `json:"updated_at"`
	} `json:"metadata"`
	Entity struct {
		Name                     string `json:"name"`
		BillingEnabled           bool   `json:"billing_enabled"`
		Status                   string `json:"status"`
		QuotaDefinitionGUID      string `json:"quota_definition_guid"`
		QuotaDefinitionURL       string `json:"quota_definition_url"`
		SpacesURL                string `json:"spaces_url"`
		DomainsURL               string `json:"domains_url"`
		PrivateDomainsURL        string `json:"private_domains_url"`
		UsersURL                 string `json:"users_url"`
		ManagersURL              string `json:"managers_url"`
		BillingManagersURL       string `json:"billing_managers_url"`
		AuditorsURL              string `json:"auditors_url"`
		AppEventsURL             string `json:"app_events_url"`
		SpaceQuotaDefinitionsURL string `json:"space_quota_definitions_url"`
	} `json:"entity"`
}
