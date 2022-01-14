package api

import (
	"fmt"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/policy-server/store"
)

type asgMapper struct {
	Marshaler marshal.Marshaler
}

func NewAsgMapper(marshaler marshal.Marshaler) AsgMapper {
	return &asgMapper{
		Marshaler: marshaler,
	}
}

func (p *asgMapper) AsBytes(storeSecurityGroups []store.SecurityGroup, pagination store.Pagination) ([]byte, error) {
	apiSecurityGroups := make([]SecurityGroup, len(storeSecurityGroups))
	for i, securityGroup := range storeSecurityGroups {
		apiSecurityGroups[i] = mapStoreSecurityGroup(securityGroup)
	}

	payload := &AsgsPayload{
		Next:           pagination.Next,
		SecurityGroups: apiSecurityGroups,
	}

	bytes, err := p.Marshaler.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %s", err)
	}
	return bytes, nil
}

func mapStoreSecurityGroup(storeSecurityGroup store.SecurityGroup) SecurityGroup {
	return SecurityGroup{
		Guid:              storeSecurityGroup.Guid,
		Name:              storeSecurityGroup.Name,
		Rules:             storeSecurityGroup.Rules,
		StagingDefault:    storeSecurityGroup.StagingDefault,
		RunningDefault:    storeSecurityGroup.RunningDefault,
		StagingSpaceGuids: storeSecurityGroup.StagingSpaceGuids,
		RunningSpaceGuids: storeSecurityGroup.RunningSpaceGuids,
	}
}
