package ovhranges

import "testing"

// TestParsePrefixes pins down what the exporter accepts as an OVH range.
// The IPv6 case matters: RIPEstat returns 42 IPv6 prefixes alongside the
// IPv4 ones, and keeping them would only waste comparisons since a
// floating IP is never IPv6. The error cases matter more: a payload this
// package cannot read must fail loudly, because a silently smaller range
// set silently narrows the check.
func TestParsePrefixes(t *testing.T) {
	cases := map[string]struct {
		rawPrefixes []string
		wantIPv4    int
		wantErr     bool
	}{
		"IPv4 prefixes are kept": {
			rawPrefixes: []string{"51.83.0.0/16", "147.135.0.0/17"},
			wantIPv4:    2,
		},
		"IPv6 prefixes are dropped": {
			rawPrefixes: []string{"51.83.0.0/16", "2001:41d0::/32"},
			wantIPv4:    1,
		},
		"an unparsable prefix is an error": {
			rawPrefixes: []string{"51.83.0.0/16", "not-a-prefix"},
			wantErr:     true,
		},
		"a payload with no IPv4 prefix at all is an error": {
			rawPrefixes: []string{"2001:41d0::/32"},
			wantErr:     true,
		},
		"an empty payload is an error": {
			rawPrefixes: nil,
			wantErr:     true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ranges, err := ParsePrefixes(tc.rawPrefixes)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %d prefixes", ranges.Len())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ranges.Len() != tc.wantIPv4 {
				t.Fatalf("got %d IPv4 prefixes, want %d", ranges.Len(), tc.wantIPv4)
			}
		})
	}
}

// TestContains covers the single decision the dangling-DNS check depends
// on. The non-IP cases are not hypothetical: Contains is fed a DNS
// record's content straight from Cloudflare, so it has to answer "not
// OVH" for anything that is not a plain IPv4 address rather than blow up.
func TestContains(t *testing.T) {
	ranges, err := ParsePrefixes([]string{"51.83.0.0/16", "147.135.0.0/17"})
	if err != nil {
		t.Fatalf("unexpected error building the ranges: %v", err)
	}

	cases := map[string]struct {
		ip   string
		want bool
	}{
		"an address inside the first range is OVH":   {ip: "51.83.12.34", want: true},
		"an address inside the second range is OVH":  {ip: "147.135.0.1", want: true},
		"an address just outside a range is not OVH": {ip: "147.135.128.1", want: false},
		"an unrelated public address is not OVH":     {ip: "8.8.8.8", want: false},
		"an IPv6 address is not OVH":                 {ip: "2001:41d0::1", want: false},
		"an empty string is not OVH":                 {ip: "", want: false},
		"a hostname is not OVH":                      {ip: "example.com", want: false},
		"an address with a port is not OVH":          {ip: "51.83.12.34:80", want: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := ranges.Contains(tc.ip); got != tc.want {
				t.Fatalf("Contains(%q) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}
}
