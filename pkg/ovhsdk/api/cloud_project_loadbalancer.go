package api

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

func GetCloudProjectRegionLoadBalancers(client *ovh.Client, projectID string, regionName string) ([]models.LoadBalancer, error) {
	var loadBalancers []models.LoadBalancer

	endpoint := "/cloud/project/" + projectID + "/region/" + regionName + "/loadbalancing/loadbalancer"

	err := client.Get(endpoint, &loadBalancers)
	if err != nil {
		return loadBalancers, err
	}

	return loadBalancers, nil
}
