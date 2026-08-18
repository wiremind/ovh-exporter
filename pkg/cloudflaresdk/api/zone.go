package api

import (
	"context"

	"github.com/cloudflare/cloudflare-go/v7/zones"
	"github.com/wiremind/ovh-exporter/pkg/cloudflaresdk/models"
)

// ListZones returns every DNS zone visible to the API token, across all pages.
func ListZones(client *zones.ZoneService) ([]models.Zone, error) {
	var result []models.Zone

	iter := client.ListAutoPaging(context.Background(), zones.ZoneListParams{})
	for iter.Next() {
		zone := iter.Current()
		result = append(result, models.Zone{ID: zone.ID, Name: zone.Name})
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
