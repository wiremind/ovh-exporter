package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"slices"
	"sync"
	"syscall"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/option"
	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	cloudflareapi "github.com/wiremind/ovh-exporter/pkg/cloudflaresdk/api"
	cloudflaremodels "github.com/wiremind/ovh-exporter/pkg/cloudflaresdk/models"
	"github.com/wiremind/ovh-exporter/pkg/ovhranges"
	"github.com/wiremind/ovh-exporter/pkg/parallel"
)

var cloudflareDanglingFloatingIPDNSInfo = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_cloudflare_dangling_floatingip_dns_info",
		Help: "Flags Cloudflare DNS A records pointing at an OVH address that no watched project currently reserves as a floating IP. Value is always 1.",
	},
	[]string{"zone", "record_name", "record_type", "ip"},
)

var cloudflareDanglingFloatingIPDNSResponding = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_cloudflare_dangling_floatingip_dns_responding",
		Help: "1 when the IP of a flagged DNS record accepts or refuses a TCP connection on one of the probed ports, so a host now holds it; 0 when every port times out.",
	},
	[]string{"zone", "record_name", "record_type", "ip"},
)

// probePorts covers SSH, HTTP, HTTPS and the Kubernetes API, the services our
// records usually front.
var probePorts = []string{"22", "80", "443", "6443"}

const (
	probeTimeout     = 2 * time.Second
	probeConcurrency = 20

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
	Responding bool
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

	// skipped collects the regions walked past on a 404 so they can be
	// reported as one line. Logging them individually costs one line per
	// macro-region per project on every cycle, which buries the rest of the
	// output; zerolog has no level configured here, so demoting them to
	// Debug would not have hidden them either.
	var skipped []string

	for _, projectID := range projectIDs {
		regions, err := cachedGetCloudProjectRegionsWithService(ovhClient, projectID, RegionServiceNetwork)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve regions for project %s: %w", projectID, err)
		}

		for _, regionName := range regions {
			floatingIPs, err := cachedGetCloudProjectRegionFloatingIPs(ovhClient, projectID, regionName)
			if err != nil {
				// Storage-only macro-regions are already filtered out by
				// RegionServiceNetwork. A 404 here means a region announces
				// the network service but has no floating IP endpoint:
				// skipped rather than aborting, so one odd region cannot
				// block the whole cross-check.
				if isOVHNotFound(err) {
					skipped = append(skipped, projectID+"/"+regionName)

					continue
				}

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

	if len(skipped) > 0 {
		logger.Info().Msgf("%d project/region pairs expose no floating IP endpoint and were skipped, %d reserved floating IPs collected", len(skipped), len(reserved))
	}

	return reserved, nil
}

// isOVHNotFound reports whether err is an OVH API 404. Only a 404 is
// tolerated by reservedFloatingIPs: anything else (a timeout, a 403 on a
// revoked credential, a 500) may hide floating IPs we do own, and a short
// set of reserved IPs turns this check into a page of false positives.
func isOVHNotFound(err error) bool {
	var apiError *ovh.APIError

	return errors.As(err, &apiError) && apiError.Code == http.StatusNotFound
}

// responds reports whether a host holds ip. A refused connection counts: only
// a live host sends the reset, an unallocated address stays silent.
func responds(ip string, ports []string, timeout time.Duration) bool {
	answers := parallel.Map(ports, len(ports), func(port string) bool {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, port), timeout)
		if err == nil {
			_ = conn.Close()

			return true
		}

		return errors.Is(err, syscall.ECONNREFUSED)
	})

	return slices.Contains(answers, true)
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

// setCloudflareDanglingFloatingIPDNSInfo is called once per finding, so the
// log line here gives the same list as the metric. It is worth the volume:
// the findings are the reason the check exists, and reading them from the
// logs needs no Prometheus.
func setCloudflareDanglingFloatingIPDNSInfo(record danglingRecord) {
	logger.Info().Msgf("dangling DNS record: %s %s -> %s (zone %s) points at an OVH address no watched project reserves as a floating IP, responding=%t", record.RecordType, record.RecordName, record.IP, record.Zone, record.Responding)

	labels := prometheus.Labels{
		"zone":        record.Zone,
		"record_name": record.RecordName,
		"record_type": record.RecordType,
		"ip":          record.IP,
	}

	cloudflareDanglingFloatingIPDNSInfo.With(labels).Set(1)

	responding := 0.0
	if record.Responding {
		responding = 1
	}

	cloudflareDanglingFloatingIPDNSResponding.With(labels).Set(responding)
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

	responding := parallel.Map(records, probeConcurrency, func(record danglingRecord) bool {
		return responds(record.IP, probePorts, probeTimeout)
	})
	for i := range records {
		records[i].Responding = responding[i]
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

	// found is filled by the fetch closure below. RefreshScope only
	// reports failure, and the two steps after the OVH walk (RIPEstat, then
	// the Cloudflare zone pagination) log nothing at all, so without this
	// count a successful check is indistinguishable from a stalled one.
	found := 0

	err := RefreshScope(
		GlobalScope{},
		CollectorCloudflareDanglingDNS,
		func() ([]danglingRecord, error) {
			records, err := fetchDanglingRecords(ctx, cfClient, ovhClient)
			found = len(records)

			return records, err
		},
		setCloudflareDanglingFloatingIPDNSInfo,
		cloudflareDanglingFloatingIPDNSInfo,
		cloudflareDanglingFloatingIPDNSResponding,
	)
	if err != nil {
		logger.Error().Msgf("failed to cross-check Cloudflare DNS records against OVH floating IPs: %v", err)

		return
	}

	logger.Info().Msgf("cross-check done, %d dangling DNS record(s) flagged", found)
}
