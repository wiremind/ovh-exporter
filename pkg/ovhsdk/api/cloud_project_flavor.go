package api

import (
	"fmt"

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

func GetCloudProjectFlavorsPerInstances(ovhClient *ovh.Client, projectID string, projectInstances []models.InstanceSummary) ([]models.Flavor, error) {
	var flavors []models.Flavor
	var apierr error = nil
	for _, instance := range projectInstances {
		if FindFlavorByID(flavors, instance.FlavorID) == nil {
			flavor, err := GetCloudProjectFlavor(ovhClient, projectID, instance.FlavorID)
			if err != nil {
				if apierr == nil {
					apierr = fmt.Errorf("Failed to retrieve flavor: %v", err)
				} else {
					apierr = fmt.Errorf("%s; %v", apierr, err)
				}
			} else {
				flavors = append(flavors, flavor)
			}
		}
	}
	return flavors, apierr
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
