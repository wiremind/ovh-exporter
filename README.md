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
export OVH_CACHE_UPDATE_INTERVAL="300"
export SERVER_PORT="8080"
```

To use the compose, add a `ovh-exporter.env` file at the root of your project with the variables filled in:

```bash
OVH_ENDPOINT="ovh-eu"
OVH_APP_KEY=""
OVH_APP_SECRET=""
OVH_CONSUMER_KEY=""
OVH_CLOUD_PROJECT_INSTANCE_BILLING_PROJECT_IDS=""
OVH_CACHE_UPDATE_INTERVAL="300"
SERVER_PORT="8080"
```

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
curl 0.0.0.0:<port>/metrics | grep "your_metric"
```

Example
```bash
curl 0.0.0.0:8080/metrics | grep "ovh_exporter_services_savingsplans_subscribed"
```