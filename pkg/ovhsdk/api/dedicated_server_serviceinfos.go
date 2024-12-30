package api

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

func GetDedicatedServerServiceInfos(client *ovh.Client, serverID string) (models.ServiceInfo, error) {
	var serviceinfos models.ServiceInfo

	endpoint := "/dedicated/server/" + serverID + "/serviceInfos"

	err := client.Get(endpoint, &serviceinfos)
	if err != nil {
		return serviceinfos, err
	}

	return serviceinfos, nil
}
