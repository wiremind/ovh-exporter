package network

import (
	"testing"

	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

func services(names ...string) []models.RegionService {
	var regionServices []models.RegionService
	for _, name := range names {
		regionServices = append(regionServices, models.RegionService{Name: name, Status: "UP"})
	}

	return regionServices
}

// TestRegionHasService uses the services OVH returned for real regions of a
// watched project: the storage-only macro-regions must be filtered out of
// the floating IP and load balancer walks, the compute regions kept.
func TestRegionHasService(t *testing.T) {
	gra := models.Region{Name: "GRA", Services: services("storage", "key-manager", "storage-s3-high-perf", "storage-s3-standard")}
	gra9 := models.Region{Name: "GRA9", Services: services("key-manager", "instance", "image", "volume", "workflow", "octavialoadbalancer", "network", "share")}
	caEastTor := models.Region{Name: "CA-EAST-TOR", Status: "DOWN", Services: services("storage-s3-high-perf", "storage-s3-standard")}
	networkDown := models.Region{Name: "GRA11", Services: []models.RegionService{{Name: "network", Status: "DOWN"}}}

	cases := map[string]struct {
		region  models.Region
		service string
		want    bool
	}{
		"storage-only region has no network":       {gra, RegionServiceNetwork, false},
		"storage-only region has no load balancer": {gra, RegionServiceLoadBalancer, false},
		"down storage-only region has no network":  {caEastTor, RegionServiceNetwork, false},
		"compute region has network":               {gra9, RegionServiceNetwork, true},
		"compute region has load balancer":         {gra9, RegionServiceLoadBalancer, true},
		"service down is still queried":            {networkDown, RegionServiceNetwork, true},
		"region without services exposes nothing":  {models.Region{Name: "X"}, RegionServiceNetwork, false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := regionHasService(tc.region, tc.service); got != tc.want {
				t.Errorf("regionHasService(%s, %s) = %v, want %v", tc.region.Name, tc.service, got, tc.want)
			}
		})
	}
}
