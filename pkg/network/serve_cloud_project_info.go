package network

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/api"
)

var cloudProjectInfo = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_cloud_project_info",
		Help: "Information about watched OVH cloud projects, value is always 1. Join on project_id to resolve the human-readable description.",
	},
	[]string{"project_id", "description", "status"},
)

func updateCloudProjectInfoPerProjectID(ovhClient *ovh.Client, projectID string) {
	logger.Info().Msgf("updating cloud project info for project %s", projectID)

	project, err := api.GetCloudProject(ovhClient, projectID)
	if err != nil {
		apiErrors.WithLabelValues("cloud_project_info").Inc()
		logger.Error().Msgf("failed to retrieve cloud project %s: %v", projectID, err)
		return
	}

	description := "undefined"
	if project.Description != nil {
		description = *project.Description
	}

	cloudProjectInfo.With(prometheus.Labels{
		"project_id":  project.ProjectID,
		"description": description,
		"status":      project.Status,
	}).Set(1)
}

func updateCloudProjectInfo(ovhClient *ovh.Client) {
	seen := make(map[string]bool)
	for _, envVar := range []string{"OVH_CLOUD_PROJECT_INSTANCE_BILLING_PROJECT_IDS", "OVH_CLOUD_PROJECT_INVENTORY_PROJECT_IDS"} {
		for _, projectID := range projectIDsFromEnv(envVar) {
			if seen[projectID] {
				continue
			}
			seen[projectID] = true

			updateCloudProjectInfoPerProjectID(ovhClient, projectID)
		}
	}
}
