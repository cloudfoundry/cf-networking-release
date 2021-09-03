package documents

type UpdateOrganizationRequest struct {
	Name                string `json:"name,omitempty"`
	Status              string `json:"status,omitempty"`
	QuotaDefinitionGUID string `json:"quota_definition_guid,omitempty"`
}
