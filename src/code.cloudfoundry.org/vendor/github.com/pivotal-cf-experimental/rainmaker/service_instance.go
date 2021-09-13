package rainmaker

import "github.com/pivotal-cf-experimental/rainmaker/internal/documents"

type ServiceInstance struct {
	GUID      string
	Name      string
	PlanGUID  string
	SpaceGUID string
}

func newServiceInstanceFromResponse(config Config, response documents.ServiceInstanceResponse) ServiceInstance {
	return ServiceInstance{
		GUID:      response.Metadata.GUID,
		Name:      response.Entity.Name,
		PlanGUID:  response.Entity.ServicePlanGUID,
		SpaceGUID: response.Entity.SpaceGUID,
	}

}
