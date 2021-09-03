package documents

import "time"

type UserResponse struct {
	Metadata struct {
		GUID      string     `json:"guid"`
		URL       string     `json:"url"`
		CreatedAt *time.Time `json:"created_at"`
		UpdatedAt *time.Time `json:"updated_at"`
	} `json:"metadata"`
	Entity struct {
		Admin                          bool   `json:"admin"`
		Active                         bool   `json:"active"`
		DefaultSpaceGUID               string `json:"default_space_guid"`
		SpacesURL                      string `json:"spaces_url"`
		OrganizationsURL               string `json:"organizations_url"`
		ManagedOrganizationsURL        string `json:"managed_organizations_url"`
		BillingManagedOrganizationsURL string `json:"billing_managed_organizations_url"`
		AuditedOrganizationsURL        string `json:"audited_organizations_url"`
		ManagedSpacesURL               string `json:"managed_spaces_url"`
		AuditedSpacesURL               string `json:"audited_spaces_url"`
	} `json:"entity"`
}
