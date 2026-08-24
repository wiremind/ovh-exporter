package network

import "github.com/prometheus/client_golang/prometheus"

// Collector names identify a collector to the apiErrors counter and to log
// messages. Kept as constants (instead of repeating the string at every
// call site) so a typo can't silently split one collector's errors across
// two different label values on ovh_exporter_api_errors_total.
const (
	CollectorCloudProjectInfo               = "cloud_project_info"
	CollectorCloudProjectInstanceBilling    = "cloud_project_instance_billing"
	CollectorCloudProjectVolume             = "cloud_project_volume"
	CollectorCloudProjectLoadBalancer       = "cloud_project_loadbalancer"
	CollectorCloudProjectFloatingIP         = "cloud_project_floatingip"
	CollectorDedicatedServerSubscription    = "dedicated_server_subscription"
	CollectorServicesSavingsPlansSubscribed = "services_savingsplans_subscribed"
	CollectorCloudflareDanglingDNS          = "cloudflare_dangling_dns"
)

// Scope identifies which label combination a RefreshScope call clears
// before repopulating it. Implementations own the actual label-key strings
// once, so call sites pass typed fields (ProjectID, Region...) instead of
// repeating raw label keys ("project_id"...) in a prometheus.Labels map at
// every call site.
type Scope interface {
	Labels() prometheus.Labels
}

// ProjectScope scopes a gauge to a single OVH cloud project.
type ProjectScope struct{ ProjectID string }

func (s ProjectScope) Labels() prometheus.Labels {
	return prometheus.Labels{"project_id": s.ProjectID}
}

// ProjectRegionScope scopes a gauge to a single region of a cloud project.
type ProjectRegionScope struct{ ProjectID, Region string }

func (s ProjectRegionScope) Labels() prometheus.Labels {
	return prometheus.Labels{"project_id": s.ProjectID, "region": s.Region}
}

// ServiceScope scopes a gauge to a single OVH service.
type ServiceScope struct{ ServiceID string }

func (s ServiceScope) Labels() prometheus.Labels {
	return prometheus.Labels{"service_id": s.ServiceID}
}

// GlobalScope clears every series on the given gauges, equivalent to
// Reset() but gated on fetch succeeding first. Use it when there is no
// natural per-item label to key off, e.g. a flat, fully-enumerated list.
type GlobalScope struct{}

func (GlobalScope) Labels() prometheus.Labels { return prometheus.Labels{} }

// RefreshScope refreshes one scope of one or more gauges from a fetch that
// may fail. On success it clears any series matching scope on every gauge
// passed in, then repopulates them via set for each fetched item. On
// failure it leaves all gauges untouched, so the previous values keep
// serving instead of a hole appearing until the next successful refresh,
// and reports the failure via apiErrors under collector.
func RefreshScope[T any](
	scope Scope,
	collector string,
	fetch func() ([]T, error),
	set func(T),
	gauges ...*prometheus.GaugeVec,
) error {
	items, err := fetch()
	if err != nil {
		apiErrors.WithLabelValues(collector).Inc()
		return err
	}

	labels := scope.Labels()
	for _, gauge := range gauges {
		gauge.DeletePartialMatch(labels)
	}
	for _, item := range items {
		set(item)
	}

	return nil
}
