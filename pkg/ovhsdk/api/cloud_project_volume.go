package api

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

func GetCloudProjectVolumes(client *ovh.Client, projectID string) ([]models.Volume, error) {
	var volumes []models.Volume

	endpoint := "/cloud/project/" + projectID + "/volume"

	err := client.Get(endpoint, &volumes)
	if err != nil {
		return volumes, err
	}

	return volumes, nil
}
