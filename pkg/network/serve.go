package network

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v3"
)

var ServeCommand = &cli.Command{
	Name:   "serve",
	Usage:  "serve all routes",
	Action: serveRoutes,
}

var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

var apiErrors = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "ovh_exporter_api_errors_total",
		Help: "Counts OVH API call failures per collector.",
	},
	[]string{"collector"},
)

func projectIDsFromEnv(envVar string) []string {
	var projectIDs []string
	for _, projectID := range strings.Split(os.Getenv(envVar), ",") {
		projectID = strings.TrimSpace(projectID)
		if projectID != "" {
			projectIDs = append(projectIDs, projectID)
		}
	}
	return projectIDs
}

func pingHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "pong")
}

func initializeMetrics() {
	prometheus.MustRegister(apiErrors)
	prometheus.MustRegister(cloudProjectInstanceBilling)
	prometheus.MustRegister(cloudProjectVolumeInfo)
	prometheus.MustRegister(cloudProjectLoadBalancerInfo)
	prometheus.MustRegister(cloudProjectFloatingIPInfo)
	prometheus.MustRegister(dedicatedServerSubscription)
	prometheus.MustRegister(dedicatedServerSubscriptionExpirationTimestamp)
	prometheus.MustRegister(servicesSavingsPlansSubscribedPlanSize)
}

func updateMetrics(ovhClient *ovh.Client) {
	cloudProjectInstanceBilling.Reset()
	updateCloudProviderInstanceBilling(ovhClient)

	cloudProjectVolumeInfo.Reset()
	updateCloudProjectVolumes(ovhClient)

	cloudProjectLoadBalancerInfo.Reset()
	updateCloudProjectLoadBalancers(ovhClient)

	cloudProjectFloatingIPInfo.Reset()
	updateCloudProjectFloatingIPs(ovhClient)

	dedicatedServerSubscription.Reset()
	dedicatedServerSubscriptionExpirationTimestamp.Reset()
	updateDedicatedServersSubscription(ovhClient)

	servicesSavingsPlansSubscribedPlanSize.Reset()
	updateAllServicesSavingsPlansSubscribed(ovhClient)
}

func setupCacheUpdater(ovhClient *ovh.Client) {
	intervalStr := os.Getenv("OVH_CACHE_UPDATE_INTERVAL")
	intervalSeconds, err := strconv.Atoi(intervalStr)
	if err != nil {
		logger.Fatal().Msgf("failed to parse OVH_CACHE_UPDATE_INTERVAL: %v", err)
	}
	cacheUpdateInterval := time.Duration(intervalSeconds) * time.Second

	logger.Info().Msgf("setting up cache updater to sync every %v", cacheUpdateInterval)

	ticker := time.NewTicker(cacheUpdateInterval)
	defer ticker.Stop()

	for {
		updateMetrics(ovhClient)

		<-ticker.C
	}
}

func serveRoutes(ctx context.Context, cmd *cli.Command) error {
	initializeMetrics()

	ovhClient, err := ovh.NewClient(
		os.Getenv("OVH_ENDPOINT"),
		os.Getenv("OVH_APP_KEY"),
		os.Getenv("OVH_APP_SECRET"),
		os.Getenv("OVH_CONSUMER_KEY"),
	)
	if err != nil {
		logger.Fatal().Msgf("failed to create OVH client: %v", err)
	}

	http.HandleFunc("/ping", pingHandler)
	http.Handle("/metrics", promhttp.Handler())

	serverPort := os.Getenv("SERVER_PORT")
	formattedServerPort := fmt.Sprintf(":%s", serverPort)
	logger.Info().Msgf("server started on port %s", formattedServerPort)

	go setupCacheUpdater(ovhClient)

	return http.ListenAndServe(formattedServerPort, nil)
}
