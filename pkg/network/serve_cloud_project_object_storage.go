package network

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/api"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

var cloudProjectObjectStorageObjects = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_cloud_project_object_storage_objects",
		Help: "Number of objects stored in an OVHcloud Object Storage bucket.",
	},
	[]string{"project_id", "region", "bucket"},
)

var cloudProjectObjectStorageBytes = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_cloud_project_object_storage_bytes",
		Help: "Number of bytes stored in an OVHcloud Object Storage bucket.",
	},
	[]string{"project_id", "region", "bucket"},
)

func setCloudProjectObjectStorageInfo(projectID string, regionName string, storageContainer models.StorageContainer) {
	labels := prometheus.Labels{
		"project_id": projectID,
		"region":     regionName,
		"bucket":     storageContainer.Name,
	}

	cloudProjectObjectStorageObjects.With(labels).Set(float64(storageContainer.ObjectsCount))
	cloudProjectObjectStorageBytes.With(labels).Set(float64(storageContainer.ObjectsSize))
}

func updateCloudProjectObjectStoragePerRegion(ovhClient *ovh.Client, projectID string, regionName string) {
	logger.Info().Msgf("updating cloud project object storage for project %s in region %s", projectID, regionName)

	err := RefreshScope(
		ProjectRegionScope{ProjectID: projectID, Region: regionName},
		CollectorCloudProjectObjectStorage,
		func() ([]models.StorageContainer, error) {
			return api.GetCloudProjectRegionStorageContainers(ovhClient, projectID, regionName)
		},
		func(storageContainer models.StorageContainer) {
			setCloudProjectObjectStorageInfo(projectID, regionName, storageContainer)
		},
		cloudProjectObjectStorageObjects, cloudProjectObjectStorageBytes,
	)
	if err != nil {
		logger.Error().Msgf("failed to retrieve object storage for project %s in region %s: %v", projectID, regionName, err)
	}
}

func updateCloudProjectObjectStoragePerProjectID(ovhClient *ovh.Client, projectID string) {
	regions, err := api.GetCloudProjectRegions(ovhClient, projectID)
	if err != nil {
		apiErrors.WithLabelValues(CollectorCloudProjectObjectStorage).Inc()
		logger.Error().Msgf("failed to retrieve regions for project %s: %v", projectID, err)
		return
	}

	for _, regionName := range regions {
		updateCloudProjectObjectStoragePerRegion(ovhClient, projectID, regionName)
	}
}

func updateCloudProjectObjectStorage(ovhClient *ovh.Client) {
	for _, projectID := range projectIDsFromEnv(EnvOVHCloudProjectInventoryProjectIDs) {
		updateCloudProjectObjectStoragePerProjectID(ovhClient, projectID)
	}
}
