package version

import (
	"code.cloudfoundry.org/cli/plugin"
)

var CurrentVersion = plugin.VersionType{
	Major: 1,
	Minor: 3,
	Build: 3,
}

type Getter struct{}

func (g *Getter) Get() plugin.VersionType {
	return CurrentVersion
}
