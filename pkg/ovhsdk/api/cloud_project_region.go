package api

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

func GetCloudProjectRegions(client *ovh.Client, projectID string) ([]string, error) {
	var regions []string

	endpoint := "/cloud/project/" + projectID + "/region"

	err := client.Get(endpoint, &regions)
	if err != nil {
		return regions, err
	}

	return regions, nil
}

func GetCloudProjectRegion(client *ovh.Client, projectID string, regionName string) (models.Region, error) {
	var region models.Region

	endpoint := "/cloud/project/" + projectID + "/region/" + regionName

	err := client.Get(endpoint, &region)
	if err != nil {
		return region, err
	}

	return region, nil
}
