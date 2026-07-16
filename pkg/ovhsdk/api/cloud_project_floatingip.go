package api

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

func GetCloudProjectRegionFloatingIPs(client *ovh.Client, projectID string, regionName string) ([]models.FloatingIP, error) {
	var floatingIPs []models.FloatingIP

	endpoint := "/cloud/project/" + projectID + "/region/" + regionName + "/floatingip"

	err := client.Get(endpoint, &floatingIPs)
	if err != nil {
		return floatingIPs, err
	}

	return floatingIPs, nil
}
