package documents

type OrganizationsListResponse struct {
	TotalResults int                    `json:"total_results"`
	TotalPages   int                    `json:"total_pages"`
	PrevURL      string                 `json:"prev_url"`
	NextURL      string                 `json:"next_url"`
	Resources    []OrganizationResponse `json:"resources"`
}
