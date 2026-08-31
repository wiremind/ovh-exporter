package network

import (
	"testing"

	cloudflaremodels "github.com/wiremind/ovh-exporter/pkg/cloudflaresdk/models"
	"github.com/wiremind/ovh-exporter/pkg/ovhranges"
)

// ovhTestRanges stands in for what RIPEstat returns for AS16276, reduced to
// two prefixes. 51.83.0.0/16 and 147.135.0.0/17 are real OVH ranges, so a
// test address inside them is representative of a floating IP.
func ovhTestRanges(t *testing.T) ovhranges.Ranges {
	t.Helper()

	ranges, err := ovhranges.ParsePrefixes([]string{"51.83.0.0/16", "147.135.0.0/17"})
	if err != nil {
		t.Fatalf("unexpected error building the test ranges: %v", err)
	}

	return ranges
}

// TestMatchDanglingRecords exercises the actual security check. A record is
// only a finding when BOTH halves hold: its target is an address OVH
// announces, AND we do not currently reserve it. The "points elsewhere"
// cases are the reason the OVH-range filter exists at all - without it the
// metric grows with the DNS estate instead of with the number of findings.
func TestMatchDanglingRecords(t *testing.T) {
	ranges := ovhTestRanges(t)

	cases := map[string]struct {
		dnsRecords  []cloudflaremodels.DNSRecord
		reservedIPs []string
		want        []danglingRecord
	}{
		"OVH address we no longer reserve is flagged": {
			dnsRecords: []cloudflaremodels.DNSRecord{
				{ZoneName: "example.com", Name: "old.example.com", Type: "A", Content: "51.83.0.99"},
			},
			reservedIPs: []string{"51.83.0.10"},
			want: []danglingRecord{
				{Zone: "example.com", RecordName: "old.example.com", RecordType: "A", IP: "51.83.0.99"},
			},
		},
		"OVH address we still reserve is not flagged": {
			dnsRecords: []cloudflaremodels.DNSRecord{
				{ZoneName: "example.com", Name: "live.example.com", Type: "A", Content: "51.83.0.10"},
			},
			reservedIPs: []string{"51.83.0.10"},
			want:        nil,
		},
		"address outside OVH is not flagged even though we don't reserve it": {
			dnsRecords: []cloudflaremodels.DNSRecord{
				{ZoneName: "example.com", Name: "aws.example.com", Type: "A", Content: "8.8.8.8"},
			},
			reservedIPs: nil,
			want:        nil,
		},
		"record with empty content is never flagged": {
			dnsRecords: []cloudflaremodels.DNSRecord{
				{ZoneName: "example.com", Name: "weird.example.com", Type: "A", Content: ""},
			},
			reservedIPs: nil,
			want:        nil,
		},
		"reserving nothing flags every OVH-hosted record but no other": {
			dnsRecords: []cloudflaremodels.DNSRecord{
				{ZoneName: "example.com", Name: "ovh.example.com", Type: "A", Content: "147.135.0.1"},
				{ZoneName: "example.com", Name: "elsewhere.example.com", Type: "A", Content: "8.8.8.8"},
			},
			reservedIPs: nil,
			want: []danglingRecord{
				{Zone: "example.com", RecordName: "ovh.example.com", RecordType: "A", IP: "147.135.0.1"},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			reserved := make(map[string]bool)
			for _, ip := range tc.reservedIPs {
				reserved[ip] = true
			}

			got := matchDanglingRecords(tc.dnsRecords, reserved, ranges)

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
