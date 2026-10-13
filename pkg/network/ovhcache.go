package network

import (
	"fmt"
	"time"

	"github.com/ovh/go-ovh/ovh"
	gocache "github.com/patrickmn/go-cache"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/api"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

var ovhAPICache = gocache.New(gocache.NoExpiration, gocache.NoExpiration)

const (
	RegionServiceNetwork      = "network"
	RegionServiceLoadBalancer = "octavialoadbalancer"
)

// regionDetailsCache keeps region details across refresh cycles
// It almost never changes so never flushed
var regionDetailsCache = gocache.New(24*time.Hour, time.Hour)

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

func cachedGetCloudProjectRegion(ovhClient *ovh.Client, projectID string, regionName string) (models.Region, error) {
	key := projectID + "/" + regionName
	if cached, found := regionDetailsCache.Get(key); found {
		return cached.(models.Region), nil
	}

	region, err := api.GetCloudProjectRegion(ovhClient, projectID, regionName)
	if err != nil {
		return models.Region{}, err
	}
	regionDetailsCache.SetDefault(key, region)

	return region, nil
}

func regionHasService(region models.Region, service string) bool {
	for _, regionService := range region.Services {
		if regionService.Name == service {
			return true
		}
	}

	return false
}

func cachedGetCloudProjectRegionsWithService(ovhClient *ovh.Client, projectID string, service string) ([]string, error) {
	regionNames, err := cachedGetCloudProjectRegions(ovhClient, projectID)
	if err != nil {
		return nil, err
	}

	var withService []string
	for _, regionName := range regionNames {
		region, err := cachedGetCloudProjectRegion(ovhClient, projectID, regionName)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve details of region %s: %w", regionName, err)
		}
		if regionHasService(region, service) {
			withService = append(withService, regionName)
		}
	}

	return withService, nil
}
