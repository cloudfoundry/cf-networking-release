package policy_client

type PoliciesV0 struct {
	TotalPolicies int        `json:"total_policies"`
	Policies      []PolicyV0 `json:"policies"`
}

type PolicyV0 struct {
	Source      SourceV0      `json:"source"`
	Destination DestinationV0 `json:"destination"`
}

type SourceV0 struct {
	ID  string `json:"id"`
	Tag string `json:"tag,omitempty"`
}

type DestinationV0 struct {
	ID       string `json:"id"`
	Tag      string `json:"tag,omitempty"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
}

type TagV0 struct {
	ID  string `json:"id"`
	Tag string `json:"tag"`
}

type SpaceV0 struct {
	Name    string `json:name`
	OrgGUID string `json:organization_guid`
}
