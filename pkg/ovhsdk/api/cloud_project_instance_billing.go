package api

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

func GetCloudProjectInstances(client *ovh.Client, projectID string) ([]models.InstanceSummary, error) {
	var instances []models.InstanceSummary

	endpoint := "/cloud/project/" + projectID + "/instance"

	err := client.Get(endpoint, &instances)
	if err != nil {
		return instances, err
	}

	return instances, nil
}

func GetCloudProjectInstance(client *ovh.Client, projectID string, instanceID string) (models.Instance, error) {
	var instanceData models.Instance

	endpoint := "/cloud/project/" + projectID + "/instance/" + instanceID

	err := client.Get(endpoint, &instanceData)
	if err != nil {
		return instanceData, err
	}

	return instanceData, nil
}
