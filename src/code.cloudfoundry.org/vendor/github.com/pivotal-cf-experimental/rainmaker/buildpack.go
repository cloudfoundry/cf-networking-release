package rainmaker

import (
	"time"

	"github.com/pivotal-cf-experimental/rainmaker/internal/documents"
)

type Buildpack struct {
	GUID      string
	URL       string
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string
	Position  int
	Enabled   bool
	Locked    bool
	Filename  string
}

func newBuildpackFromResponse(config Config, response documents.BuildpackResponse) Buildpack {
	return Buildpack{
		GUID:      response.Metadata.GUID,
		URL:       response.Metadata.URL,
		CreatedAt: response.Metadata.CreatedAt,
		Name:      response.Entity.Name,
		Position:  response.Entity.Position,
		Enabled:   response.Entity.Enabled,
		Locked:    response.Entity.Locked,
		Filename:  response.Entity.Filename,
	}
}
