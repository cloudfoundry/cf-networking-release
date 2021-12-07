package store

type SpaceSecurityGroupRules struct {
	SpaceGuid    string
	StagingRules string
	RunningRules string
}

type SecurityGroup struct {
	Guid              string
	Name              string
	Rules             string
	StagingDefault    bool
	RunningDefault    bool
	StagingSpaceGuids []string
	RunningSpaceGuids []string
}
