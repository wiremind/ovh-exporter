package api

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

func FindFlavorByID(flavors []models.Flavor, flavorID string) *models.Flavor {
	for _, flavor := range flavors {
		if flavor.ID == flavorID {
			return &flavor
		}
	}
	return nil
}

func GetCloudProjectFlavor(client *ovh.Client, projectID string, flavorID string) (models.Flavor, error) {
	var flavor models.Flavor

	endpoint := "/cloud/project/" + projectID + "/flavor/" + flavorID

	err := client.Get(endpoint, &flavor)
	if err != nil {
		return flavor, err
	}

	return flavor, nil
}
