package network

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/api"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

var cloudProjectLoadBalancerInfo = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_cloud_project_loadbalancer_info",
		Help: "Tracks OVH cloud project load balancers with their name and statuses.",
	},
	[]string{"project_id", "region", "loadbalancer_id", "loadbalancer_name", "provisioning_status", "operating_status", "vip_address"},
)

func setCloudProjectLoadBalancerInfo(projectID string, regionName string, loadBalancer models.LoadBalancer) {
	cloudProjectLoadBalancerInfo.With(prometheus.Labels{
		"project_id":          projectID,
		"region":              regionName,
		"loadbalancer_id":     loadBalancer.ID,
		"loadbalancer_name":   loadBalancer.Name,
		"provisioning_status": loadBalancer.ProvisioningStatus,
		"operating_status":    loadBalancer.OperatingStatus,
		"vip_address":         loadBalancer.VipAddress,
	}).Set(1)
}

func updateCloudProjectLoadBalancersPerRegion(ovhClient *ovh.Client, projectID string, regionName string) {
	logger.Info().Msgf("updating cloud project load balancers for project %s in region %s", projectID, regionName)

	loadBalancers, err := api.GetCloudProjectRegionLoadBalancers(ovhClient, projectID, regionName)
	if err != nil {
		apiErrors.WithLabelValues("cloud_project_loadbalancer").Inc()
		logger.Error().Msgf("failed to retrieve load balancers for project %s in region %s: %v", projectID, regionName, err)
		return
	}

	for _, loadBalancer := range loadBalancers {
		setCloudProjectLoadBalancerInfo(projectID, regionName, loadBalancer)
	}
}

func updateCloudProjectLoadBalancersPerProjectID(ovhClient *ovh.Client, projectID string) {
	regions, err := api.GetCloudProjectRegions(ovhClient, projectID)
	if err != nil {
		apiErrors.WithLabelValues("cloud_project_loadbalancer").Inc()
		logger.Error().Msgf("failed to retrieve regions for project %s: %v", projectID, err)
		return
	}

	for _, regionName := range regions {
		updateCloudProjectLoadBalancersPerRegion(ovhClient, projectID, regionName)
	}
}

func updateCloudProjectLoadBalancers(ovhClient *ovh.Client) {
	for _, projectID := range projectIDsFromEnv("OVH_CLOUD_PROJECT_INVENTORY_PROJECT_IDS") {
		updateCloudProjectLoadBalancersPerProjectID(ovhClient, projectID)
	}
}
