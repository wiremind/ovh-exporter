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

// cloudflareDanglingFloatingIPDNSInfo flags Cloudflare DNS records that
// still resolve to an OVH floating IP we hold but that is no longer attached
// to any instance or gateway. As long as the record exists, whoever manages
// to (re-)reserve that same floating IP on OVH — us later, or an attacker if
// it ever gets released — starts receiving the traffic the DNS name still
// sends there. This is a dangling-DNS / subdomain-takeover setup, so any
// series on this gauge is a finding to act on: delete the DNS record, or
// release the floating IP if it truly serves no purpose.
var cloudflareDanglingFloatingIPDNSInfo = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_cloudflare_dangling_floatingip_dns_info",
		Help: "Flags Cloudflare DNS records pointing at an OVH floating IP that is reserved but not attached to anything. Value is always 1; any series here is a dangling-DNS finding.",
	},
	[]string{"zone", "record_name", "record_type", "ip", "floatingip_id", "project_id", "region"},
)

// unusedFloatingIP is a floating IP found reserved but unattached
// (AssociatedEntity == nil), indexed by IP address so a DNS record's content
// can be looked up directly.
type unusedFloatingIP struct {
	FloatingIPID string
	ProjectID    string
	Region       string
}

// addUnusedFloatingIPs scans one project/region's floating IPs and records
// into unused every one that is reserved but currently unattached
// (AssociatedEntity == nil), keyed by IP address. It is a pure function
// (no OVH call, no global state) so the matching rule can be unit-tested
// without a live client.
//
// OVH's API response is untyped JSON decoded straight into models.FloatingIP
// (see pkg/ovhsdk/models): a future field rename, a null where a string was
// expected, or a genuinely empty IP would decode as Go zero values rather
// than fail the call. An empty IP is therefore ignored here instead of
// blindly indexed — matching on "" against a Cloudflare record with equally
// empty content would otherwise fabricate a finding out of two unrelated
// blanks.
func addUnusedFloatingIPs(unused map[string]unusedFloatingIP, floatingIPs []models.FloatingIP, projectID string, region string) {
	for _, floatingIP := range floatingIPs {
		if floatingIP.AssociatedEntity != nil {
			continue
		}
		if floatingIP.IP == "" {
			continue
		}
		unused[floatingIP.IP] = unusedFloatingIP{FloatingIPID: floatingIP.ID, ProjectID: projectID, Region: region}
	}
}

// collectUnusedFloatingIPs walks every region of every watched cloud project
// and returns the floating IPs that are reserved but currently unattached,
// keyed by IP address. A failure on one project/region is reported to
// apiErrors and skipped rather than aborting the whole scan, so one broken
// project can't hide dangling DNS on the others.
func collectUnusedFloatingIPs(ovhClient *ovh.Client) map[string]unusedFloatingIP {
	unused := make(map[string]unusedFloatingIP)

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

			addUnusedFloatingIPs(unused, floatingIPs, projectID, regionName)
		}
	}

	return unused
}

// danglingRecord is a Cloudflare DNS record found pointing at an unused
// floating IP, carrying enough OVH context to act on the finding.
type danglingRecord struct {
	Zone         string
	RecordName   string
	RecordType   string
	IP           string
	FloatingIPID string
	ProjectID    string
	Region       string
}

// matchDanglingRecords is the actual security check: it flags every DNS
// record whose content matches a floating IP OVH still reserves for us but
// no longer attaches to anything. Kept as a pure function, independent of
// both the Cloudflare and OVH clients, so this matching rule — the one
// piece of logic that actually has to be correct — can be unit-tested
// directly instead of only through a live end-to-end run.
//
// dnsRecord.Content is untyped data from Cloudflare's API, same caveat as
// OVH's: an empty content (a record type this exporter doesn't expect, or a
// future schema change) is skipped rather than matched against a floating
// IP that was itself skipped for being empty.
func matchDanglingRecords(dnsRecords []cloudflaremodels.DNSRecord, unused map[string]unusedFloatingIP) []danglingRecord {
	var records []danglingRecord

	for _, dnsRecord := range dnsRecords {
		if dnsRecord.Content == "" {
			continue
		}

		found, ok := unused[dnsRecord.Content]
		if !ok {
			continue
		}

		records = append(records, danglingRecord{
			Zone:         dnsRecord.ZoneName,
			RecordName:   dnsRecord.Name,
			RecordType:   dnsRecord.Type,
			IP:           dnsRecord.Content,
			FloatingIPID: found.FloatingIPID,
			ProjectID:    found.ProjectID,
			Region:       found.Region,
		})
	}

	return records
}

func setCloudflareDanglingFloatingIPDNSInfo(record danglingRecord) {
	cloudflareDanglingFloatingIPDNSInfo.With(prometheus.Labels{
		"zone":          record.Zone,
		"record_name":   record.RecordName,
		"record_type":   record.RecordType,
		"ip":            record.IP,
		"floatingip_id": record.FloatingIPID,
		"project_id":    record.ProjectID,
		"region":        record.Region,
	}).Set(1)
}

// updateCloudflareDanglingFloatingIPDNS cross-checks every Cloudflare DNS
// A record we can see against the OVH floating IPs currently unattached. It
// uses GlobalScope: unlike the other collectors there is no natural
// per-item label to scope a partial refresh on, since a finding is defined
// by matching two independent APIs (OVH and Cloudflare) rather than by
// enumerating a single list.
func updateCloudflareDanglingFloatingIPDNS(ovhClient *ovh.Client, cfClient *cloudflare.Client) {
	logger.Info().Msg("cross-checking Cloudflare DNS records against unused OVH floating IPs")

	unusedFloatingIPs := collectUnusedFloatingIPs(ovhClient)

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

				records = append(records, matchDanglingRecords(dnsRecords, unusedFloatingIPs)...)
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
