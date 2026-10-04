package network

import (
	"strings"

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

func updateCloudProjectObjectStoragePerRegion(ovhClient *ovh.Client, projectID string, regionName string) bool {
	logger.Info().Msgf("updating cloud project object storage for project %s in region %s", projectID, regionName)

	skipped := false

	err := RefreshScope(
		ProjectRegionScope{ProjectID: projectID, Region: regionName},
		CollectorCloudProjectObjectStorage,
		func() ([]models.StorageContainer, error) {
			storageContainers, err := api.GetCloudProjectRegionStorageContainers(ovhClient, projectID, regionName)
			if isOVHNotFound(err) {
				skipped = true

				return nil, nil
			}

			return storageContainers, err
		},
		func(storageContainer models.StorageContainer) {
			setCloudProjectObjectStorageInfo(projectID, regionName, storageContainer)
		},
		cloudProjectObjectStorageObjects, cloudProjectObjectStorageBytes,
	)
	if err != nil {
		logger.Error().Msgf("failed to retrieve object storage for project %s in region %s: %v", projectID, regionName, err)
	}

	return skipped
}

func updateCloudProjectObjectStoragePerProjectID(ovhClient *ovh.Client, projectID string) {
	regions, err := cachedGetCloudProjectRegions(ovhClient, projectID)
	if err != nil {
		apiErrors.WithLabelValues(CollectorCloudProjectObjectStorage).Inc()
		logger.Error().Msgf("failed to retrieve regions for project %s: %v", projectID, err)
		return
	}

	var skipped []string

	for _, regionName := range regions {
		if updateCloudProjectObjectStoragePerRegion(ovhClient, projectID, regionName) {
			skipped = append(skipped, regionName)
		}
	}

	if len(skipped) > 0 {
		logger.Info().Msgf("project %s hosts no object storage in %d of its %d regions, skipped: %s", projectID, len(skipped), len(regions), strings.Join(skipped, ", "))
	}
}

func updateCloudProjectObjectStorage(ovhClient *ovh.Client) {
	for _, projectID := range projectIDsFromEnv(EnvOVHCloudProjectInventoryProjectIDs) {
		updateCloudProjectObjectStoragePerProjectID(ovhClient, projectID)
	}
}
