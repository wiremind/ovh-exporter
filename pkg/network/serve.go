package network

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/urfave/cli"
)

var ServeCommand = cli.Command{
	Name:   "serve",
	Usage:  "serve all routes",
	Action: serveRoutes,
}

var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

func pingHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "pong")
}

func initializeMetrics() {
	prometheus.MustRegister(cloudProjectInstanceBilling)
	prometheus.MustRegister(dedicatedServerSubscription)
	prometheus.MustRegister(dedicatedServerSubscriptionExpirationTimestamp)
	prometheus.MustRegister(servicesSavingsPlansSubscribedPlanSize)
}

func updateMetrics(ovhClient *ovh.Client) {
	cloudProjectInstanceBilling.Reset()
	updateCloudProviderInstanceBilling(ovhClient)

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

func serveRoutes(clicontext *cli.Context) error {
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
