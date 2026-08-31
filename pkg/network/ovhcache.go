package network

import (
	"github.com/ovh/go-ovh/ovh"
	gocache "github.com/patrickmn/go-cache"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/api"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

// ovhAPICache memoizes the OVH calls that more than one collector needs,
// so a second collector reading the same inventory costs no extra API
// call. It is flushed at the start of every refresh cycle by
// updateMetrics: entries must never survive a cycle, or a collector would
// export the previous cycle's inventory.
var ovhAPICache = gocache.New(gocache.NoExpiration, gocache.NoExpiration)

func cachedGetCloudProjectRegions(ovhClient *ovh.Client, projectID string) ([]string, error) {
	key := "regions/" + projectID
	if cached, found := ovhAPICache.Get(key); found {
		return cached.([]string), nil
	}

	regions, err := api.GetCloudProjectRegions(ovhClient, projectID)
	if err != nil {
		return nil, err
	}
	ovhAPICache.Set(key, regions, gocache.NoExpiration)

	return regions, nil
}

func cachedGetCloudProjectRegionFloatingIPs(ovhClient *ovh.Client, projectID string, regionName string) ([]models.FloatingIP, error) {
	key := "floatingips/" + projectID + "/" + regionName
	if cached, found := ovhAPICache.Get(key); found {
		return cached.([]models.FloatingIP), nil
	}

	floatingIPs, err := api.GetCloudProjectRegionFloatingIPs(ovhClient, projectID, regionName)
	if err != nil {
		return nil, err
	}
	ovhAPICache.Set(key, floatingIPs, gocache.NoExpiration)

	return floatingIPs, nil
}
