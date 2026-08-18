package network

import (
	cloudflare "github.com/cloudflare/cloudflare-go/v7"
	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	cloudflareapi "github.com/wiremind/ovh-exporter/pkg/cloudflaresdk/api"
	cloudflaremodels "github.com/wiremind/ovh-exporter/pkg/cloudflaresdk/models"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/api"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

// cloudflareDanglingFloatingIPDNSInfo flags Cloudflare DNS A records whose
// target IP is not currently reserved as an OVH floating IP by any of the
// watched projects. Floating IPs are drawn from a pool OVH shares across
// customers: if a DNS record points at one we don't hold, whoever reserves
// that exact address next on OVH starts receiving the traffic the record
// still sends there. Any series here is a finding to act on: fix or delete
// the DNS record, or re-reserve the floating IP if it's actually still
// needed.
//
// This only checks OVH's side of the equation: a record can also show up
// here simply because it legitimately points somewhere other than OVH
// (another cloud, on-prem...). This exporter has no way to tell that case
// apart from an actually released floating IP without knowing which OVH
// address ranges are involved, so expect noise from records that were
// never meant to resolve to an OVH floating IP.
var cloudflareDanglingFloatingIPDNSInfo = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_cloudflare_dangling_floatingip_dns_info",
		Help: "Flags Cloudflare DNS A records whose IP is not currently reserved as an OVH floating IP by any watched project. Value is always 1.",
	},
	[]string{"zone", "record_name", "record_type", "ip"},
)

// addReservedFloatingIPs records into reserved every IP from floatingIPs,
// attached or not: as long as OVH still reports it as ours, nobody else on
// OVH can reserve that exact address. It is a pure function (no OVH call,
// no global state) so the matching rule can be unit-tested without a live
// client.
//
// OVH's response is untyped JSON decoded straight into models.FloatingIP
// (see pkg/ovhsdk/models): a future field rename or a null where a string
// was expected would decode as a Go zero value rather than fail the call.
// An empty IP is therefore ignored here instead of blindly indexed - it
// would otherwise make every Cloudflare record with equally empty content
// look "reserved" by coincidence.
func addReservedFloatingIPs(reserved map[string]bool, floatingIPs []models.FloatingIP) {
	for _, floatingIP := range floatingIPs {
		if floatingIP.IP == "" {
			continue
		}
		reserved[floatingIP.IP] = true
	}
}

// collectReservedFloatingIPs walks every region of every watched cloud
// project and returns the set of floating IPs currently reserved by us,
// attached or not. A failure on one project/region is reported to
// apiErrors and skipped rather than aborting the whole scan, so one broken
// project can't hide dangling DNS on the others.
func collectReservedFloatingIPs(ovhClient *ovh.Client) map[string]bool {
	reserved := make(map[string]bool)

	for _, projectID := range projectIDsFromEnv(EnvOVHCloudProjectInventoryProjectIDs) {
		regions, err := api.GetCloudProjectRegions(ovhClient, projectID)
		if err != nil {
			apiErrors.WithLabelValues(CollectorCloudflareDanglingDNS).Inc()
			logger.Error().Msgf("failed to retrieve regions for project %s: %v", projectID, err)
			continue
		}

		for _, regionName := range regions {
			floatingIPs, err := api.GetCloudProjectRegionFloatingIPs(ovhClient, projectID, regionName)
			if err != nil {
				apiErrors.WithLabelValues(CollectorCloudflareDanglingDNS).Inc()
				logger.Error().Msgf("failed to retrieve floating IPs for project %s in region %s: %v", projectID, regionName, err)
				continue
			}

			addReservedFloatingIPs(reserved, floatingIPs)
		}
	}

	return reserved
}

// danglingRecord is a Cloudflare DNS record found pointing at an IP not
// currently reserved as an OVH floating IP.
type danglingRecord struct {
	Zone       string
	RecordName string
	RecordType string
	IP         string
}

// matchDanglingRecords is the actual security check: it flags every DNS
// record whose content is not one of the floating IPs we currently reserve
// on OVH. Kept as a pure function, independent of both the Cloudflare and
// OVH clients, so this matching rule can be unit-tested directly instead of
// only through a live end-to-end run.
//
// dnsRecord.Content is untyped data from Cloudflare's API, same caveat as
// OVH's: an empty content (a record type this exporter doesn't expect, or a
// future schema change) is skipped rather than flagged as dangling.
func matchDanglingRecords(dnsRecords []cloudflaremodels.DNSRecord, reservedFloatingIPs map[string]bool) []danglingRecord {
	var records []danglingRecord

	for _, dnsRecord := range dnsRecords {
		if dnsRecord.Content == "" {
			continue
		}
		if reservedFloatingIPs[dnsRecord.Content] {
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

// updateCloudflareDanglingFloatingIPDNS cross-checks every Cloudflare DNS A
// record we can see against the OVH floating IPs currently reserved by us.
// It uses GlobalScope: unlike the other collectors there is no natural
// per-item label to scope a partial refresh on, since a finding is defined
// by matching two independent APIs (OVH and Cloudflare) rather than by
// enumerating a single list.
func updateCloudflareDanglingFloatingIPDNS(ovhClient *ovh.Client, cfClient *cloudflare.Client) {
	logger.Info().Msg("cross-checking Cloudflare DNS records against OVH floating IPs")

	reservedFloatingIPs := collectReservedFloatingIPs(ovhClient)

	err := RefreshScope(
		GlobalScope{},
		CollectorCloudflareDanglingDNS,
		func() ([]danglingRecord, error) {
			zones, err := cloudflareapi.ListZones(cfClient.Zones)
			if err != nil {
				return nil, err
			}

			var records []danglingRecord
			for _, zone := range zones {
				dnsRecords, err := cloudflareapi.ListZoneARecords(cfClient.DNS.Records, zone.ID, zone.Name)
				if err != nil {
					return nil, err
				}

				records = append(records, matchDanglingRecords(dnsRecords, reservedFloatingIPs)...)
			}

			return records, nil
		},
		setCloudflareDanglingFloatingIPDNSInfo,
		cloudflareDanglingFloatingIPDNSInfo,
	)
	if err != nil {
		logger.Error().Msgf("failed to cross-check Cloudflare DNS records against OVH floating IPs: %v", err)
	}
}
