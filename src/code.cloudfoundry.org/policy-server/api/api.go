package api

//go:generate counterfeiter -generate

import "code.cloudfoundry.org/policy-server/store"

var ICMPDefault = -1
var AppLifecycleDefault = "all"

//counterfeiter:generate -o fakes/policy_mapper.go --fake-name PolicyMapper . PolicyMapper
type PolicyMapper interface {
	AsStorePolicy([]byte) ([]store.Policy, error) // unmarshal
	AsBytes([]store.Policy) ([]byte, error)       // marshal
}

//counterfeiter:generate -o fakes/asg_mapper.go --fake-name AsgMapper . AsgMapper
type AsgMapper interface {
	AsBytes([]store.SecurityGroup, store.Pagination) ([]byte, error) // marshal
}

type PoliciesPayload struct {
	TotalPolicies int      `json:"total_policies"`
	Policies      []Policy `json:"policies"`
}

type Policy struct {
	Source      Source      `json:"source"`
	Destination Destination `json:"destination"`
}

type Source struct {
	ID   string `json:"id"`
	Tag  string `json:"tag,omitempty"`
	Type string `json:"type,omitempty"`
}

type Destination struct {
	ID       string    `json:"id"`
	Tag      string    `json:"tag,omitempty"`
	Protocol string    `json:"protocol"`
	Ports    Ports     `json:"ports"`
	Type     string    `json:"type,omitempty"`
	IPs      []IPRange `json:"ips,omitempty"`
}

type IPRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Ports struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type Tag struct {
	ID   string `json:"id"`
	Tag  string `json:"tag"`
	Type string `json:"type"`
}

type AsgsPayload struct {
	Next           int             `json:"next"`
	SecurityGroups []SecurityGroup `json:"security_groups"`
}

type SecurityGroup struct {
	Guid              string   `json:"guid"`
	Name              string   `json:"name"`
	Rules             string   `json:"rules"`
	StagingDefault    bool     `json:"staging_default"`
	RunningDefault    bool     `json:"running_default"`
	StagingSpaceGuids []string `json:"staging_space_guids"`
	RunningSpaceGuids []string `json:"running_space_guids"`
}
