package models

type CloudProject struct {
	ProjectID   string  `json:"project_id"`
	ProjectName *string `json:"projectName"` // Use *string for nullable string
	Description *string `json:"description"` // Use *string for nullable string
	PlanCode    string  `json:"planCode"`
	Status      string  `json:"status"` // Allowed: "creating", "deleted", "deleting", "ok", "suspended"
}
