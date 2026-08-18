package network

import (
	"testing"

	cloudflaremodels "github.com/wiremind/ovh-exporter/pkg/cloudflaresdk/models"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

// TestAddReservedFloatingIPs pins down the one rule the whole dangling-DNS
// check is built on: any floating IP OVH still reports as ours - attached
// or not - counts as reserved, since nobody else on OVH can claim it while
// we hold it. It also guards against OVH's response decoding to Go zero
// values instead of erroring on a schema we don't expect (see the
// empty-IP case) - untrusted external JSON, not something this repo
// controls the shape of.
func TestAddReservedFloatingIPs(t *testing.T) {
	cases := map[string]struct {
		floatingIPs []models.FloatingIP
		want        map[string]bool
	}{
		"unattached floating IP still counts as reserved": {
			floatingIPs: []models.FloatingIP{
				{ID: "fip-1", IP: "203.0.113.10", AssociatedEntity: nil},
			},
			want: map[string]bool{"203.0.113.10": true},
		},
		"attached floating IP counts as reserved too": {
			floatingIPs: []models.FloatingIP{
				{ID: "fip-2", IP: "203.0.113.20", AssociatedEntity: &models.FloatingIPAssociatedEntity{ID: "instance-1"}},
			},
			want: map[string]bool{"203.0.113.20": true},
		},
		"floating IP with an empty IP is skipped": {
			floatingIPs: []models.FloatingIP{
				{ID: "fip-3", IP: "", AssociatedEntity: nil},
			},
			want: map[string]bool{},
		},
		"multiple floating IPs are all recorded": {
			floatingIPs: []models.FloatingIP{
				{ID: "fip-4", IP: "203.0.113.40", AssociatedEntity: &models.FloatingIPAssociatedEntity{ID: "instance-2"}},
				{ID: "fip-5", IP: "203.0.113.50", AssociatedEntity: nil},
			},
			want: map[string]bool{"203.0.113.40": true, "203.0.113.50": true},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			reserved := make(map[string]bool)

			addReservedFloatingIPs(reserved, tc.floatingIPs)

			if len(reserved) != len(tc.want) {
				t.Fatalf("got %v, want %v", reserved, tc.want)
			}
			for ip, want := range tc.want {
				if got := reserved[ip]; got != want {
					t.Fatalf("reserved[%q] = %v, want %v", ip, got, want)
				}
			}
		})
	}
}

// TestMatchDanglingRecords exercises the actual security check: a DNS
// record is only a finding when its content is NOT one of the floating IPs
// we currently reserve on OVH, and an empty content (an unexpected
// Cloudflare record shape) must never be treated as dangling.
func TestMatchDanglingRecords(t *testing.T) {
	reserved := map[string]bool{"203.0.113.10": true}

	cases := map[string]struct {
		dnsRecords []cloudflaremodels.DNSRecord
		reserved   map[string]bool
		want       []danglingRecord
	}{
		"record pointing at an IP we still reserve is not flagged": {
			dnsRecords: []cloudflaremodels.DNSRecord{
				{ZoneName: "example.com", Name: "live.example.com", Type: "A", Content: "203.0.113.10"},
			},
			reserved: reserved,
			want:     nil,
		},
		"record pointing at an IP we don't reserve is flagged": {
			dnsRecords: []cloudflaremodels.DNSRecord{
				{ZoneName: "example.com", Name: "old.example.com", Type: "A", Content: "203.0.113.99"},
			},
			reserved: reserved,
			want: []danglingRecord{
				{Zone: "example.com", RecordName: "old.example.com", RecordType: "A", IP: "203.0.113.99"},
			},
		},
		"record with empty content is never flagged": {
			dnsRecords: []cloudflaremodels.DNSRecord{
				{ZoneName: "example.com", Name: "weird.example.com", Type: "A", Content: ""},
			},
			reserved: map[string]bool{},
			want:     nil,
		},
		"no reserved floating IPs at all flags every record": {
			dnsRecords: []cloudflaremodels.DNSRecord{
				{ZoneName: "example.com", Name: "old.example.com", Type: "A", Content: "203.0.113.10"},
			},
			reserved: map[string]bool{},
			want: []danglingRecord{
				{Zone: "example.com", RecordName: "old.example.com", RecordType: "A", IP: "203.0.113.10"},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := matchDanglingRecords(tc.dnsRecords, tc.reserved)

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
