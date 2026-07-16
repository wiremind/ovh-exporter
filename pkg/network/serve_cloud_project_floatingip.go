package network

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/api"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

var cloudProjectFloatingIPInfo = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_cloud_project_floatingip_info",
		Help: "Tracks OVH cloud project floating IPs with their status and associated entity.",
	},
	[]string{"project_id", "region", "floatingip_id", "ip", "network_id", "status", "associated_entity_id", "associated_entity_type", "associated_entity_ip"},
)

func setCloudProjectFloatingIPInfo(projectID string, regionName string, floatingIP models.FloatingIP) {
	associatedEntityID := ""
	associatedEntityType := ""
	associatedEntityIP := ""
	if floatingIP.AssociatedEntity != nil {
		associatedEntityID = floatingIP.AssociatedEntity.ID
		associatedEntityType = floatingIP.AssociatedEntity.Type
		associatedEntityIP = floatingIP.AssociatedEntity.IP
	}

	cloudProjectFloatingIPInfo.With(prometheus.Labels{
		"project_id":             projectID,
		"region":                 regionName,
		"floatingip_id":          floatingIP.ID,
		"ip":                     floatingIP.IP,
		"network_id":             floatingIP.NetworkID,
		"status":                 floatingIP.Status,
		"associated_entity_id":   associatedEntityID,
		"associated_entity_type": associatedEntityType,
		"associated_entity_ip":   associatedEntityIP,
	}).Set(1)
}

func updateCloudProjectFloatingIPsPerRegion(ovhClient *ovh.Client, projectID string, regionName string) {
	logger.Info().Msgf("updating cloud project floating IPs for project %s in region %s", projectID, regionName)

	floatingIPs, err := api.GetCloudProjectRegionFloatingIPs(ovhClient, projectID, regionName)
	if err != nil {
		apiErrors.WithLabelValues("cloud_project_floatingip").Inc()
		logger.Error().Msgf("failed to retrieve floating IPs for project %s in region %s: %v", projectID, regionName, err)
		return
	}

	for _, floatingIP := range floatingIPs {
		setCloudProjectFloatingIPInfo(projectID, regionName, floatingIP)
	}
}

func updateCloudProjectFloatingIPsPerProjectID(ovhClient *ovh.Client, projectID string) {
	regions, err := api.GetCloudProjectRegions(ovhClient, projectID)
	if err != nil {
		apiErrors.WithLabelValues("cloud_project_floatingip").Inc()
		logger.Error().Msgf("failed to retrieve regions for project %s: %v", projectID, err)
		return
	}

	for _, regionName := range regions {
		updateCloudProjectFloatingIPsPerRegion(ovhClient, projectID, regionName)
	}
}

func updateCloudProjectFloatingIPs(ovhClient *ovh.Client) {
	for _, projectID := range projectIDsFromEnv("OVH_CLOUD_PROJECT_INVENTORY_PROJECT_IDS") {
		updateCloudProjectFloatingIPsPerProjectID(ovhClient, projectID)
	}
}
