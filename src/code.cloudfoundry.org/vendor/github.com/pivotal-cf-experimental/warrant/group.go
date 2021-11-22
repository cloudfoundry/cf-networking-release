package warrant

import (
	"time"

	"github.com/pivotal-cf-experimental/warrant/internal/documents"
)

// Group is the representation of a group resource within UAA.
type Group struct {
	// ID is the unique identifier for the group resource.
	ID string

	// DisplayName is the human-friendly name given to a group.
	DisplayName string

	// Description is the human readable description of the group.
	Description string

	// Version is an integer value indicating which revision this resource represents.
	Version int

	// CreatedAt is a timestamp value indicating when the group was created.
	CreatedAt time.Time

	// UpdatedAt is a timestamp value indicating when the group was last modified.
	UpdatedAt time.Time

	// Members is the list of members to be included in the group.
	Members []Member
}

func newGroupFromResponse(config Config, response documents.GroupResponse) Group {
	var members []Member
	for _, member := range response.Members {
		members = append(members, Member{
			Type:   member.Type,
			Value:  member.Value,
			Origin: member.Origin,
		})
	}

	return Group{
		ID:          response.ID,
		Description: response.Description,
		DisplayName: response.DisplayName,
		Members:     members,
		Version:     response.Meta.Version,
		CreatedAt:   response.Meta.Created,
		UpdatedAt:   response.Meta.LastModified,
	}
}
