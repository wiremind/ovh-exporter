# ovh-exporter

Prometheus exporter for the OVH API.

## Configuration

To generate a unique link on the [OVH Api Portal](https://www.ovh.com/auth/api/createToken) with the correct OVH permissions needed for this project, run:

```bash
ovh-exporter credentials
```

Alternatively, you can use Go:

```go
go run cmd/ovh-exporter/main.go credentials
```

Once you have the credentials, create a `ovh-exporter.env` file containing the following variables:

```bash
export OVH_ENDPOINT="ovh-eu"
export OVH_APP_KEY=""
export OVH_APP_SECRET=""
export OVH_CONSUMER_KEY=""
export OVH_CLOUD_PROJECT_INSTANCE_BILLING_PROJECT_IDS=""
export OVH_CLOUD_PROJECT_INVENTORY_PROJECT_IDS=""
export OVH_DEDICATED_SERVER_SUBSCRIPTION_ENABLED="true"
export OVH_CACHE_UPDATE_INTERVAL="300"
export SERVER_PORT="8080"
export CLOUDFLARE_API_TOKEN=""
```

To use the compose, add a `ovh-exporter.env` file at the root of your project with the variables filled in:

```bash
OVH_ENDPOINT="ovh-eu"
OVH_APP_KEY=""
OVH_APP_SECRET=""
OVH_CONSUMER_KEY=""
OVH_CLOUD_PROJECT_INSTANCE_BILLING_PROJECT_IDS=""
OVH_CLOUD_PROJECT_INVENTORY_PROJECT_IDS=""
OVH_DEDICATED_SERVER_SUBSCRIPTION_ENABLED="true"
OVH_CACHE_UPDATE_INTERVAL="300"
SERVER_PORT="8080"
CLOUDFLARE_API_TOKEN=""
```

The projects' id can be found in the `Public Cloud` tab of OVH console.

### Cloudflare dangling-DNS check (optional)

If `CLOUDFLARE_API_TOKEN` is set, ovh-exporter also cross-checks Cloudflare
DNS against OVH. A DNS `A` record is flagged when **both** hold:

1. its IP is inside an address range OVH announces to the Internet, and
2. no project listed in `OVH_CLOUD_PROJECT_INVENTORY_PROJECT_IDS` currently
   reserves that IP as a floating IP.

Floating IPs come from a pool OVH shares across customers, so whoever
reserves that exact address next starts receiving the traffic the DNS name
still sends there. This is exposed as
`ovh_exporter_cloudflare_dangling_floatingip_dns_info`. A series on it is a
record to act on: fix or delete the DNS record, or re-reserve the floating
IP if it's still needed.

Condition 1 is what keeps the metric a finding list rather than a dump of
the whole DNS estate: without it, every record legitimately pointing
somewhere other than OVH (another cloud, on-prem, a SaaS target) would be
flagged too. The OVH ranges are read once a day from RIPEstat, the RIPE
NCC's public service, using OVH's AS number (`AS16276`, the identifier an
operator is known by in Internet routing):

```bash
curl -s "https://stat.ripe.net/data/announced-prefixes/data.json?resource=AS16276" | jq -r '.data.prefixes[].prefix'
```

No credentials are involved, and the exporter needs outbound HTTPS to
`stat.ripe.net` for the check to work.

Some noise remains by design: an OVH address that is not a floating IP of
ours - a dedicated server, an additional IP, another OVH customer's
address we point at on purpose - is indistinguishable from a released
floating IP from the outside, and is reported.

The check reads the projects listed in
`OVH_CLOUD_PROJECT_INVENTORY_PROJECT_IDS`, so that variable must be set
too. If it is empty, or if any OVH or RIPEstat call fails, the refresh is
skipped and the previous values keep being served: an incomplete list of
reserved floating IPs would otherwise flag every OVH-hosted record at once.
Failures are counted on `ovh_exporter_api_errors_total{collector="cloudflare_dangling_dns"}`,
which is the series to alert on to catch a check that has silently stopped
updating.

Leave `CLOUDFLARE_API_TOKEN` empty to disable the check entirely.

The token needs, scoped to the zones you want checked (or "All zones"):

- `Zone` → `Zone` → `Read`
- `Zone` → `DNS` → `Read`

## Running

### Running the Binary

To run the exporter, execute the following command:

```bash
ovh-exporter serve
```

### Running with Compose

If you prefer using the compose, use:

```bash
nerdctl compose up
```

## Developer Guide

### Adding New Metrics

Follow these steps to add new metrics:

1. **Add the required routes** in `pkg/credentials/generate` for the OVH API Token.
2. **Add the API calls** in `ovhsdk/api`. Create a new file for each route.
3. **Define the models** in `ovhsdk/models` based on the schema from the API responses.
4. **Create the metric** in `pkg/networks` and write the necessary custom code.
5. **Update the initialization functions** in `pkg/network/serve.go` by adding your functions to `initializeMetrics()` and `updateMetrics()`.

Once you've added the metric, test it by running the Compose file. If needed, set up port forwarding, and then run the following command:

```bash
curl -s 0.0.0.0:<port>/metrics | grep "your_metric"
```

Example
```bash
curl -s 0.0.0.0:8080/metrics | grep "ovh_exporter_services_savingsplans_subscribed"
```