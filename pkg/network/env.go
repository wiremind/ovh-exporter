package network

// Environment variable names read by the exporter. Kept as constants
// instead of repeating the string at every os.Getenv call site so a typo
// can't silently create a second, always-empty variable (e.g. reading
// "OVH_CLOUD_PROJECT_INVENTORY_PROJECT_ID" somewhere would compile fine but
// never resolve to any project).
const (
	// EnvOVHEndpoint, EnvOVHAppKey, EnvOVHAppSecret and EnvOVHConsumerKey are
	// the go-ovh client credentials, see `ovh-exporter credentials`.
	EnvOVHEndpoint    = "OVH_ENDPOINT"
	EnvOVHAppKey      = "OVH_APP_KEY"
	EnvOVHAppSecret   = "OVH_APP_SECRET"
	EnvOVHConsumerKey = "OVH_CONSUMER_KEY"

	// EnvOVHCloudProjectInstanceBillingProjectIDs and
	// EnvOVHCloudProjectInventoryProjectIDs are comma-separated lists of OVH
	// cloud project IDs, read by projectIDsFromEnv. They are two separate
	// variables because billing and inventory collectors don't necessarily
	// watch the same set of projects.
	EnvOVHCloudProjectInstanceBillingProjectIDs = "OVH_CLOUD_PROJECT_INSTANCE_BILLING_PROJECT_IDS"
	EnvOVHCloudProjectInventoryProjectIDs       = "OVH_CLOUD_PROJECT_INVENTORY_PROJECT_IDS"

	// EnvOVHDedicatedServerSubscriptionEnabled disables the dedicated server
	// subscription collector when set to "false".
	EnvOVHDedicatedServerSubscriptionEnabled = "OVH_DEDICATED_SERVER_SUBSCRIPTION_ENABLED"

	// EnvOVHCacheUpdateInterval is the refresh interval, in seconds, between
	// two full metric collections.
	EnvOVHCacheUpdateInterval = "OVH_CACHE_UPDATE_INTERVAL"

	// EnvServerPort is the port the HTTP server listens on.
	EnvServerPort = "SERVER_PORT"

	// EnvCloudflareAPIToken enables the Cloudflare dangling floating IP DNS
	// check when set. Left empty, the check is skipped entirely.
	EnvCloudflareAPIToken = "CLOUDFLARE_API_TOKEN"
)
