package documents

type CreateUserRequest struct {
	GUID             string `json:"guid"`
	DefaultSpaceGUID string `json:"default_space_guid,omitempty"`
}
