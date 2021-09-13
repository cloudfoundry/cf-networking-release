package documents

type CreateOrganizationRequest struct {
	GUID                string `json:"guid"`
	Name                string `json:"name"`
	Status              string `json:"status,omitempty"`
	QuotaDefinitionGUID string `json:"quota_definition_guid,omitempty"`
}
