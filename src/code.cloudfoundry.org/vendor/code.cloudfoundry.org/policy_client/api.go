package policy_client

import (
	"encoding/json"
	"strconv"
)

type Policies struct {
	TotalPolicies int      `json:"total_policies"`
	Policies      []Policy `json:"policies"`
}

type Policy struct {
	Source      Source      `json:"source"`
	Destination Destination `json:"destination"`
}

type SecurityGroup struct {
	Guid              string             `json:"guid"`
	Name              string             `json:"name"`
	Rules             SecurityGroupRules `json:"rules"`
	StagingDefault    bool               `json:"staging_default"`
	RunningDefault    bool               `json:"running_default"`
	StagingSpaceGuids []string           `json:"staging_space_guids"`
	RunningSpaceGuids []string           `json:"running_space_guids"`
}

type IPRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Source struct {
	ID  string `json:"id"`
	Tag string `json:"tag,omitempty"`
}

type Destination struct {
	ID       string `json:"id"`
	Tag      string `json:"tag,omitempty"`
	Protocol string `json:"protocol"`
	Ports    Ports  `json:"ports"`
}

type Ports struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type Tag struct {
	ID  string `json:"id"`
	Tag string `json:"tag"`
}

type Space struct {
	Name    string `json:"name"`
	OrgGUID string `json:"organization_guid"`
}

type SecurityGroupRules []SecurityGroupRule

type SecurityGroupRule struct {
	Protocol    string `json:"protocol"`
	Destination string `json:"destination"`
	Ports       string `json:"ports,omitempty"`
	Type        int    `json:"type"`
	Code        int    `json:"code"`
	Description string `json:"description,omitempty"`
	Log         bool   `json:"log"`
}

func (sgr *SecurityGroupRules) UnmarshalJSON(data []byte) error {
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return err

	}
	type securityGroups SecurityGroupRules
	if err := json.Unmarshal([]byte(s), (*securityGroups)(sgr)); err != nil {
		return err
	}

	return nil
}
