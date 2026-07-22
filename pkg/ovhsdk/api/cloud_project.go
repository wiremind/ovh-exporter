package api

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

func GetCloudProject(client *ovh.Client, projectID string) (models.CloudProject, error) {
	var project models.CloudProject

	endpoint := "/cloud/project/" + projectID

	err := client.Get(endpoint, &project)
	if err != nil {
		return project, err
	}

	return project, nil
}
