package documents

type ApplicationResponse struct {
	Metadata struct {
		GUID string
	}
	Entity struct {
		Name      string
		SpaceGUID string `json:"space_guid"`
		Diego     bool
	}
}
