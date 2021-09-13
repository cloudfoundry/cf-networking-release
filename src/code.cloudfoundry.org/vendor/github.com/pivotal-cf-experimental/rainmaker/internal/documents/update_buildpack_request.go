package documents

type UpdateBuildpackRequest struct {
	Name     *string `json:"name,omitempty"`
	Position *int    `json:"position,omitempty"`
	Enabled  *bool   `json:"enabled,omitempty"`
	Locked   *bool   `json:"locked,omitempty"`
	Filename *string `json:"filename,omitempty"`
}
