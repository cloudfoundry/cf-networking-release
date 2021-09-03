package documents

import (
	"encoding/json"
	"time"
)

// Meta represents the JSON transport data structure of
// the metadata describing a resource.
type Meta struct {
	// Version is the version of the resource.
	Version int `json:"version"`

	// Created is a timestamp value indicating when the
	// resource was created.
	Created time.Time `json:"created"`

	// LastModified is a timestamp value indicating the most
	// recent time at which the resource was updated.
	LastModified time.Time `json:"lastModified"`
}

// MarshalJSON converts the Meta struct into a JSON representation.
func (m Meta) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"version":      m.Version,
		"created":      m.Created.Format("2006-01-02T15:04:05.000Z"),
		"lastModified": m.LastModified.Format("2006-01-02T15:04:05.000Z"),
	})
} // TODO: UAA team is investigating this hack as a possible bug
