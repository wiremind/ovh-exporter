package api

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

func GetCloudProjectInstances(client *ovh.Client, projectID string) ([]models.InstanceSummary, error) {
	// Define the response structure for the instance billing
	var instances []models.InstanceSummary

	// Build the API endpoint
	endpoint := "/cloud/project/" + projectID + "/instance"

	// Call OVH API to get instance data
	err := client.Get(endpoint, &instances)
	if err != nil {
		return instances, err
	}

	// Return the billing price for this instance
	return instances, nil
}

func GetCloudProjectInstance(client *ovh.Client, projectID string, instanceID string) (models.Instance, error) {
	// Define the response structure for the instance billing
	var instanceData models.Instance

	// Build the API endpoint
	endpoint := "/cloud/project/" + projectID + "/instance/" + instanceID

	// Call OVH API to get instance data
	err := client.Get(endpoint, &instanceData)
	if err != nil {
		return instanceData, err
	}

	// Return the billing price for this instance
	return instanceData, nil
}
