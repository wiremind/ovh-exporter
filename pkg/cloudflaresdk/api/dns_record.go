package api

import (
	"context"

	cloudflare "github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/dns"
	"github.com/wiremind/ovh-exporter/pkg/cloudflaresdk/models"
)

// ListZoneARecords returns every A record of the given zone, across all
// pages. Only A records are fetched: OVH floating IPs are IPv4 addresses,
// so no other record type (AAAA, CNAME...) can point to one.
func ListZoneARecords(client *dns.RecordService, zoneID string, zoneName string) ([]models.DNSRecord, error) {
	var result []models.DNSRecord

	params := dns.RecordListParams{
		ZoneID: cloudflare.F(zoneID),
		Type:   cloudflare.F(dns.RecordListParamsTypeA),
	}

	iter := client.ListAutoPaging(context.Background(), params)
	for iter.Next() {
		record := iter.Current()
		result = append(result, models.DNSRecord{
			ZoneID:   zoneID,
			ZoneName: zoneName,
			Name:     record.Name,
			Type:     string(record.Type),
			Content:  record.Content,
		})
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
