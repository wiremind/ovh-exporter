package network

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/option"
	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	cloudflareapi "github.com/wiremind/ovh-exporter/pkg/cloudflaresdk/api"
	cloudflaremodels "github.com/wiremind/ovh-exporter/pkg/cloudflaresdk/models"
	"github.com/wiremind/ovh-exporter/pkg/ovhranges"
)

var cloudflareDanglingFloatingIPDNSInfo = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_cloudflare_dangling_floatingip_dns_info",
		Help: "Flags Cloudflare DNS A records pointing at an OVH address that no watched project currently reserves as a floating IP. Value is always 1.",
	},
	[]string{"zone", "record_name", "record_type", "ip"},
)

const (
	// cloudflareRequestTimeout bounds a single Cloudflare HTTP call. The
	// list endpoints are paginated, so a per-call timeout is the only one
	// the SDK can enforce.
	cloudflareRequestTimeout = 15 * time.Second

	// danglingDNSCheckTimeout bounds the whole cross-check, pagination
	// included, so a stalled walk cannot outlive the refresh cycle.
	danglingDNSCheckTimeout = 5 * time.Minute
)

// cloudflareClient builds the Cloudflare client once, on the first refresh.
// It returns nil when the token is unset, which disables the check.
var cloudflareClient = sync.OnceValue(func() *cloudflare.Client {
	apiToken := os.Getenv(EnvCloudflareAPIToken)
	if apiToken == "" {
		logger.Info().Msgf("%s not set, disabling the Cloudflare dangling floating IP DNS check", EnvCloudflareAPIToken)
		return nil
	}

	return cloudflare.NewClient(
		option.WithAPIToken(apiToken),
		option.WithRequestTimeout(cloudflareRequestTimeout),
	)
})

type danglingRecord struct {
	Zone       string
	RecordName string
	RecordType string
	IP         string
}

// reservedFloatingIPs returns every floating IP the watched projects hold,
// attached or not: as long as OVH reports one as ours, nobody else can
// reserve that exact address.
//
// The OVH calls it makes are the same ones updateCloudProjectFloatingIPs
// already made earlier in the cycle, and both go through ovhAPICache, so
// this walk costs no extra API call.
//
// Any error aborts instead of returning a partial set: the set is the only
// thing keeping a DNS record from being flagged, so a missing project does
// not degrade the check, it inverts it into a page of false positives.
func reservedFloatingIPs(ovhClient *ovh.Client) (map[string]bool, error) {
	projectIDs := projectIDsFromEnv(EnvOVHCloudProjectInventoryProjectIDs)
	if len(projectIDs) == 0 {
		return nil, fmt.Errorf("no project in %s, every OVH-hosted DNS record would look dangling", EnvOVHCloudProjectInventoryProjectIDs)
	}

	reserved := make(map[string]bool)
	for _, projectID := range projectIDs {
		regions, err := cachedGetCloudProjectRegions(ovhClient, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve regions for project %s: %w", projectID, err)
		}

		for _, regionName := range regions {
			floatingIPs, err := cachedGetCloudProjectRegionFloatingIPs(ovhClient, projectID, regionName)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve floating IPs for project %s in region %s: %w", projectID, regionName, err)
			}

			for _, floatingIP := range floatingIPs {
				if floatingIP.IP == "" {
					continue
				}
				reserved[floatingIP.IP] = true
			}
		}
	}

	return reserved, nil
}

func matchDanglingRecords(dnsRecords []cloudflaremodels.DNSRecord, reserved map[string]bool, ovhRanges ovhranges.Ranges) []danglingRecord {
	var records []danglingRecord

	for _, dnsRecord := range dnsRecords {
		if !ovhRanges.Contains(dnsRecord.Content) {
			continue
		}
		if reserved[dnsRecord.Content] {
			continue
		}

		records = append(records, danglingRecord{
			Zone:       dnsRecord.ZoneName,
			RecordName: dnsRecord.Name,
			RecordType: dnsRecord.Type,
			IP:         dnsRecord.Content,
		})
	}

	return records
}

func setCloudflareDanglingFloatingIPDNSInfo(record danglingRecord) {
	cloudflareDanglingFloatingIPDNSInfo.With(prometheus.Labels{
		"zone":        record.Zone,
		"record_name": record.RecordName,
		"record_type": record.RecordType,
		"ip":          record.IP,
	}).Set(1)
}

func fetchDanglingRecords(ctx context.Context, cfClient *cloudflare.Client, ovhClient *ovh.Client) ([]danglingRecord, error) {
	reserved, err := reservedFloatingIPs(ovhClient)
	if err != nil {
		return nil, err
	}

	ovhRanges, err := ovhranges.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve OVH announced address ranges: %w", err)
	}

	zones, err := cloudflareapi.ListZones(ctx, cfClient.Zones)
	if err != nil {
		return nil, err
	}

	var records []danglingRecord
	for _, zone := range zones {
		dnsRecords, err := cloudflareapi.ListZoneARecords(ctx, cfClient.DNS.Records, zone.ID, zone.Name)
		if err != nil {
			return nil, err
		}

		records = append(records, matchDanglingRecords(dnsRecords, reserved, ovhRanges)...)
	}

	return records, nil
}

// updateCloudflareDanglingFloatingIPDNS cross-checks every Cloudflare DNS A
// record we can see against the OVH floating IPs the watched projects
// currently reserve, restricted to addresses OVH announces.
//
// It uses GlobalScope: unlike the other collectors there is no natural
// per-item label to scope a partial refresh on, since a finding is defined
// by matching two independent APIs (OVH and Cloudflare) rather than by
// enumerating a single list.
func updateCloudflareDanglingFloatingIPDNS(ovhClient *ovh.Client) {
	cfClient := cloudflareClient()
	if cfClient == nil {
		return
	}

	logger.Info().Msg("cross-checking Cloudflare DNS records against OVH floating IPs")

	ctx, cancel := context.WithTimeout(context.Background(), danglingDNSCheckTimeout)
	defer cancel()

	err := RefreshScope(
		GlobalScope{},
		CollectorCloudflareDanglingDNS,
		func() ([]danglingRecord, error) { return fetchDanglingRecords(ctx, cfClient, ovhClient) },
		setCloudflareDanglingFloatingIPDNSInfo,
		cloudflareDanglingFloatingIPDNSInfo,
	)
	if err != nil {
		logger.Error().Msgf("failed to cross-check Cloudflare DNS records against OVH floating IPs: %v", err)
	}
}
