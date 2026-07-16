package network

import (
	"slices"
	"strconv"
	"strings"

	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/api"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

var cloudProjectVolumeInfo = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_cloud_project_volume_info",
		Help: "Tracks OVH cloud project volumes with their name, status and attachment.",
	},
	[]string{"project_id", "volume_id", "volume_name", "region", "status", "volume_type", "size_gb", "attached_to"},
)

func setCloudProjectVolumeInfo(projectID string, volume models.Volume) {
	attachedTo := slices.Clone(volume.AttachedTo)
	slices.Sort(attachedTo)

	cloudProjectVolumeInfo.With(prometheus.Labels{
		"project_id":  projectID,
		"volume_id":   volume.ID,
		"volume_name": volume.Name,
		"region":      volume.Region,
		"status":      volume.Status,
		"volume_type": volume.Type,
		"size_gb":     strconv.Itoa(volume.Size),
		"attached_to": strings.Join(attachedTo, ","),
	}).Set(1)
}

func updateCloudProjectVolumesPerProjectID(ovhClient *ovh.Client, projectID string) {
	logger.Info().Msgf("updating cloud project volumes for project %s", projectID)

	volumes, err := api.GetCloudProjectVolumes(ovhClient, projectID)
	if err != nil {
		apiErrors.WithLabelValues("cloud_project_volume").Inc()
		logger.Error().Msgf("failed to retrieve volumes for project %s: %v", projectID, err)
		return
	}

	for _, volume := range volumes {
		setCloudProjectVolumeInfo(projectID, volume)
	}
}

func updateCloudProjectVolumes(ovhClient *ovh.Client) {
	for _, projectID := range projectIDsFromEnv("OVH_CLOUD_PROJECT_INVENTORY_PROJECT_IDS") {
		updateCloudProjectVolumesPerProjectID(ovhClient, projectID)
	}
}
