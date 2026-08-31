package models

// Zone is a Cloudflare DNS zone, trimmed to the fields this exporter needs.
type Zone struct {
	ID   string
	Name string
}

// DNSRecord is a Cloudflare DNS record, trimmed to the fields this exporter
// needs to cross-check against OVH floating IPs. Content holds the record's
// target: an IPv4 address for an A record.
type DNSRecord struct {
	ZoneID   string
	ZoneName string
	Name     string
	Type     string
	Content  string
}
