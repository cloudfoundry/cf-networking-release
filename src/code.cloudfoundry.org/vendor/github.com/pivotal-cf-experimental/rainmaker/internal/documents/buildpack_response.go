package documents

import "time"

type BuildpackResponse struct {
	Metadata struct {
		GUID      string    `json:"guid"`
		URL       string    `json:"url"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"metadata"`
	Entity struct {
		Name     string `json:"name"`
		Position int    `json:"position"`
		Enabled  bool   `json:"enabled"`
		Locked   bool   `json:"locked"`
		Filename string `json:"filename"`
	} `json:"entity"`
}
