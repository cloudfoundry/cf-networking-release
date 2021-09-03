package documents

type CreateSpaceRequest struct {
	GUID             string `json:"guid"`
	Name             string `json:"name"`
	OrganizationGUID string `json:"organization_guid"`
}
