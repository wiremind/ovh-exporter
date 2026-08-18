package network

import (
	"testing"

	cloudflaremodels "github.com/wiremind/ovh-exporter/pkg/cloudflaresdk/models"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

// TestAddUnusedFloatingIPs pins down the one rule the whole dangling-DNS
// check is built on: a floating IP only counts as "unused" when it is
// attached to nothing. It also guards against OVH's response decoding to
// Go zero values instead of erroring on a schema we don't expect (see the
// empty-IP case) — untrusted external JSON, not something this repo
// controls the shape of.
func TestAddUnusedFloatingIPs(t *testing.T) {
	cases := map[string]struct {
		floatingIPs []models.FloatingIP
		want        map[string]unusedFloatingIP
	}{
		"unattached floating IP is recorded as unused": {
			floatingIPs: []models.FloatingIP{
				{ID: "fip-1", IP: "203.0.113.10", AssociatedEntity: nil},
			},
			want: map[string]unusedFloatingIP{
				"203.0.113.10": {FloatingIPID: "fip-1", ProjectID: "proj-1", Region: "GRA"},
			},
		},
		"attached floating IP is not recorded": {
			floatingIPs: []models.FloatingIP{
				{ID: "fip-2", IP: "203.0.113.20", AssociatedEntity: &models.FloatingIPAssociatedEntity{ID: "instance-1"}},
			},
			want: map[string]unusedFloatingIP{},
		},
		"floating IP with an empty IP is skipped even when unattached": {
			floatingIPs: []models.FloatingIP{
				{ID: "fip-3", IP: "", AssociatedEntity: nil},
			},
			want: map[string]unusedFloatingIP{},
		},
		"mix of attached and unattached keeps only the unattached one": {
			floatingIPs: []models.FloatingIP{
				{ID: "fip-4", IP: "203.0.113.40", AssociatedEntity: &models.FloatingIPAssociatedEntity{ID: "instance-2"}},
				{ID: "fip-5", IP: "203.0.113.50", AssociatedEntity: nil},
			},
			want: map[string]unusedFloatingIP{
				"203.0.113.50": {FloatingIPID: "fip-5", ProjectID: "proj-1", Region: "GRA"},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			unused := make(map[string]unusedFloatingIP)

			addUnusedFloatingIPs(unused, tc.floatingIPs, "proj-1", "GRA")

			if len(unused) != len(tc.want) {
				t.Fatalf("got %v, want %v", unused, tc.want)
			}
			for ip, want := range tc.want {
				if got := unused[ip]; got != want {
					t.Fatalf("unused[%q] = %v, want %v", ip, got, want)
				}
			}
		})
	}
}

// TestMatchDanglingRecords exercises the actual security check: a DNS
// record is only a finding when its content matches an IP marked unused,
// and an empty content (an unexpected Cloudflare record shape) must never
// be treated as a match just because an equally empty floating IP was
// skipped upstream.
func TestMatchDanglingRecords(t *testing.T) {
	unused := map[string]unusedFloatingIP{
		"203.0.113.10": {FloatingIPID: "fip-1", ProjectID: "proj-1", Region: "GRA"},
	}

	cases := map[string]struct {
		dnsRecords []cloudflaremodels.DNSRecord
		unused     map[string]unusedFloatingIP
		want       []danglingRecord
	}{
		"record pointing at an unused floating IP is flagged": {
			dnsRecords: []cloudflaremodels.DNSRecord{
				{ZoneName: "example.com", Name: "old.example.com", Type: "A", Content: "203.0.113.10"},
			},
			unused: unused,
			want: []danglingRecord{
				{Zone: "example.com", RecordName: "old.example.com", RecordType: "A", IP: "203.0.113.10", FloatingIPID: "fip-1", ProjectID: "proj-1", Region: "GRA"},
			},
		},
		"record pointing at an IP we still use is not flagged": {
			dnsRecords: []cloudflaremodels.DNSRecord{
				{ZoneName: "example.com", Name: "live.example.com", Type: "A", Content: "203.0.113.99"},
			},
			unused: unused,
			want:   nil,
		},
		"record with empty content is never matched": {
			dnsRecords: []cloudflaremodels.DNSRecord{
				{ZoneName: "example.com", Name: "weird.example.com", Type: "A", Content: ""},
			},
			unused: map[string]unusedFloatingIP{"": {FloatingIPID: "fip-x"}},
			want:   nil,
		},
		"no unused floating IPs means no findings": {
			dnsRecords: []cloudflaremodels.DNSRecord{
				{ZoneName: "example.com", Name: "old.example.com", Type: "A", Content: "203.0.113.10"},
			},
			unused: map[string]unusedFloatingIP{},
			want:   nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := matchDanglingRecords(tc.dnsRecords, tc.unused)

			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got[%d] = %v, want %v", i, got[i], tc.want[i])
				}
			}
		})
	}
}
