package store

type SecurityGroup struct {
	Guid              string
	Name              string
	Rules             string
	StagingDefault    bool
	RunningDefault    bool
	StagingSpaceGuids []string
	RunningSpaceGuids []string
}

type SpaceGuid string

type SpaceSecurityGroupRules struct {
	StagingRules []SecurityGroup
	RunningRules []SecurityGroup
}

type SecurityGroupRulesBySpace map[SpaceGuid]SpaceSecurityGroupRules
