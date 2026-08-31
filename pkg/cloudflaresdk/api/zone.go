package api

import (
	"context"

	"github.com/cloudflare/cloudflare-go/v7/zones"
	"github.com/wiremind/ovh-exporter/pkg/cloudflaresdk/models"
)

// ListZones returns every DNS zone visible to the API token, across all
// pages. ctx is the caller's, so a shutdown cuts the walk short instead of
// holding the process open until the last page comes back.
func ListZones(ctx context.Context, client *zones.ZoneService) ([]models.Zone, error) {
	var result []models.Zone

	iter := client.ListAutoPaging(ctx, zones.ZoneListParams{})
	for iter.Next() {
		zone := iter.Current()
		result = append(result, models.Zone{ID: zone.ID, Name: zone.Name})
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
