# ovh-exporter
Prometheus exporter for the OVH API

# Configuration

Run this command to generate a unique link with correct ovh permissions needed for this project
```bash
ovh-exporter credentials
```

Then source a .env file containing these variables
```bash
export OVH_ENDPOINT="ovh-eu"
export OVH_APP_KEY=""
export OVH_APP_SECRET=""
export OVH_CONSUMER_KEY=""
export OVH_CLOUD_PROJECT_INSTANCE_BILLING_PROJECT_ID=""
export OVH_CACHE_UPDATE_INTERVAL="60"
export SERVER_PORT="8080"
```

# Running

## Binary
```bash
ovh-exporter serve
```

## Compose file
```bash
nerdctl compose up
```
