package documents

type CreateBuildpackRequest struct {
	Name     string  `json:"name"`
	Position *int    `json:"position,omitempty"`
	Enabled  *bool   `json:"enabled,omitempty"`
	Locked   *bool   `json:"locked,omitempty"`
	Filename *string `json:"filename,omitempty"`
}
