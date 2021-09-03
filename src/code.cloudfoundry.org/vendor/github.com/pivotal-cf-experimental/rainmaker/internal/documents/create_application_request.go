package documents

type CreateApplicationRequest struct {
	GUID      string `json:"guid"` // TODO: this seems like a mistake...
	Name      string `json:"name"`
	SpaceGUID string `json:"space_guid"`
	Diego     bool   `json:"diego"`
}
