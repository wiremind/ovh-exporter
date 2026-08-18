// Package ovhranges answers a single question: is this IPv4 address inside
// an address range OVH announces to the Internet?
//
// It is what turns the dangling-DNS check from a list of every DNS record
// not pointing at one of our floating IPs (i.e. most of the DNS estate)
// into a list of records pointing at an OVH address we no longer hold.
// Only the latter is a finding: an OVH address we released can be
// reserved by another OVH customer, who then receives the traffic the
// record still sends there. A record pointing at AWS, at on-prem or at a
// SaaS target is none of this exporter's business.
package ovhranges

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// ovhASN is OVH's Autonomous System number: the identifier an operator is
// known by in Internet routing. Every address OVH hosting serves is
// announced under it, which is why one lookup covers all of their ranges
// without this exporter having to maintain a hardcoded list.
//
// OVH also owns AS35540 (OVH Telecom), deliberately left out: that is
// their consumer ISP business, whose addresses are end-user access lines,
// never a floating IP we could have reserved.
const ovhASN = "AS16276"

// ripeStatAnnouncedPrefixesURL returns the prefixes an ASN currently
// announces. RIPEstat is the RIPE NCC's own public service: no
// authentication, no API key, one request for the whole list.
const ripeStatAnnouncedPrefixesURL = "https://stat.ripe.net/data/announced-prefixes/data.json?resource=" + ovhASN

const fetchTimeout = 15 * time.Second

// ripeStatResponse is the subset of RIPEstat's payload this package reads.
type ripeStatResponse struct {
	Status string `json:"status"`
	Data   struct {
		Prefixes []struct {
			Prefix string `json:"prefix"`
		} `json:"prefixes"`
	} `json:"data"`
}

// Ranges is a set of IPv4 prefixes, ready to be tested against an address.
type Ranges struct {
	prefixes []netip.Prefix
}

func (r Ranges) Len() int { return len(r.prefixes) }

func (r Ranges) Contains(ip string) bool {
	address, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	if !address.Is4() {
		return false
	}

	for _, prefix := range r.prefixes {
		if prefix.Contains(address) {
			return true
		}
	}

	return false
}

func ParsePrefixes(rawPrefixes []string) (Ranges, error) {
	var prefixes []netip.Prefix

	for _, rawPrefix := range rawPrefixes {
		prefix, err := netip.ParsePrefix(rawPrefix)
		if err != nil {
			return Ranges{}, fmt.Errorf("failed to parse prefix %q: %w", rawPrefix, err)
		}
		if !prefix.Addr().Is4() {
			continue
		}
		prefixes = append(prefixes, prefix)
	}

	if len(prefixes) == 0 {
		return Ranges{}, fmt.Errorf("no IPv4 prefix in the %d prefixes returned for %s", len(rawPrefixes), ovhASN)
	}

	return Ranges{prefixes: prefixes}, nil
}

// fetch retrieves OVH's currently announced IPv4 ranges from RIPEstat.
func fetch(ctx context.Context) (Ranges, error) {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, ripeStatAnnouncedPrefixesURL, nil)
	if err != nil {
		return Ranges{}, fmt.Errorf("failed to build the RIPEstat request: %w", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return Ranges{}, fmt.Errorf("failed to call RIPEstat: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return Ranges{}, fmt.Errorf("RIPEstat returned HTTP %d", response.StatusCode)
	}

	var payload ripeStatResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return Ranges{}, fmt.Errorf("failed to decode the RIPEstat response: %w", err)
	}

	// RIPEstat answers 200 with a status field of its own, so a degraded
	// answer has to be caught here rather than by the HTTP code alone.
	if payload.Status != "ok" {
		return Ranges{}, fmt.Errorf("RIPEstat reported status %q", payload.Status)
	}

	rawPrefixes := make([]string, 0, len(payload.Data.Prefixes))
	for _, prefix := range payload.Data.Prefixes {
		rawPrefixes = append(rawPrefixes, prefix.Prefix)
	}

	return ParsePrefixes(rawPrefixes)
}

// cacheTTL is deliberately long: an operator's announced ranges move on a
// scale of months, and RIPEstat is a free public service. Caching also
// keeps a RIPEstat outage from failing every caller until it is over.
const cacheTTL = 24 * time.Hour

const cacheKey = "announced-prefixes"

// cache holds the single value this package serves. It is a package-level
// singleton on purpose: the ranges are a property of OVH, not of a caller,
// so nothing is gained by making every caller carry its own cache around.
var cache = gocache.New(cacheTTL, cacheTTL)

// Get returns OVH's announced IPv4 ranges, calling RIPEstat only when the
// cached value is missing or older than cacheTTL.
func Get(ctx context.Context) (Ranges, error) {
	if cached, found := cache.Get(cacheKey); found {
		return cached.(Ranges), nil
	}

	ranges, err := fetch(ctx)
	if err != nil {
		return Ranges{}, err
	}
	cache.Set(cacheKey, ranges, gocache.DefaultExpiration)

	return ranges, nil
}
