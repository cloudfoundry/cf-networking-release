package documents

type ServiceInstanceResponse struct {
	Metadata struct {
		GUID      string `json:"guid"`
		URL       string `json:"url"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	} `json:"metadata"`
	Entity struct {
		Name            string            `json:"name"`
		Credentials     map[string]string `json:"credentials"`
		ServicePlanGUID string            `json:"service_plan_guid"`
		SpaceGUID       string            `json:"space_guid"`
		GatewayData     string            `json:"gateway_data"`
		DashboardURL    string            `json:"dashboard_url"`
		Type            string            `json:"type"`
		LastOperation   struct {
			Type        string `json:"type"`
			State       string `json:"state"`
			Description string `json:"description"`
			UpdatedAt   string `json:"updated_at"`
		} `json:"last_operation"`
		SpaceURL           string `json:"space_url"`
		ServicePlanURL     string `json:"service_plan_url"`
		ServiceBindingsURL string `json:"service_bindings_url"`
	} `json:"entity"`
}
