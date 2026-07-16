package api

import (
	"github.com/ovh/go-ovh/ovh"
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
