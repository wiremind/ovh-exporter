package api

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

func GetDedicatedServers(client *ovh.Client) ([]models.Server, error) {
	var serviceNames []string
	var servers []models.Server

	endpoint := "/dedicated/server"

	err := client.Get(endpoint, &serviceNames)
	if err != nil {
		return servers, err
	}

	for _, serviceName := range serviceNames {
		servers = append(servers, models.Server{ID: serviceName})
	}

	return servers, nil
}
