package documents

type CreateServiceInstanceRequest struct {
	Name      string `json:"name"`
	PlanGUID  string `json:"service_plan_guid"`
	SpaceGUID string `json:"space_guid"`
}
