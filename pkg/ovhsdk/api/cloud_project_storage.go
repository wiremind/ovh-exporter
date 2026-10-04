package api

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

func GetCloudProjectRegionStorageContainers(client *ovh.Client, projectID string, regionName string) ([]models.StorageContainer, error) {
	var storageContainers []models.StorageContainer

	endpoint := "/cloud/project/" + projectID + "/region/" + regionName + "/storage"

	err := client.Get(endpoint, &storageContainers)
	if err != nil {
		return storageContainers, err
	}

	return storageContainers, nil
}
