package network

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/urfave/cli"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/api"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

var ServeCommand = cli.Command{
	Name:   "serve",
	Usage:  "serve all routes",
	Action: serveRoutes,
}

var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

var cloudProjectInstanceBilling = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_cloud_project_instance_billing",
		Help: "Tracks the billing for OVH cloud project instances.",
	},
	[]string{"project_id", "instance_id", "instance_name", "billing_type", "billing_code", "billing_monthly_date", "billing_monthly_status"},
)

func pingHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "pong")
}

func initializeMetrics() {
	prometheus.MustRegister(cloudProjectInstanceBilling)
}

func setCloudProjectInstanceBilling(projectID string, instanceID string, instanceName string, billingType string, billingCode string, billingMonthlyDate time.Time, billingMonthlyStatus string, amount float64) {
	billingMonthlyDateFormatted := billingMonthlyDate.Format("2006-01-02 15:04:05")

	cloudProjectInstanceBilling.With(prometheus.Labels{
		"project_id":             projectID,
		"instance_id":            instanceID,
		"instance_name":          instanceName,
		"billing_type":           billingType,
		"billing_code":           billingCode,
		"billing_monthly_date":   billingMonthlyDateFormatted,
		"billing_monthly_status": billingMonthlyStatus,
	}).Set(amount)
}

func updateCloudProviderInstanceBilling(ovhClient *ovh.Client) {
	logger.Info().Msg("updating cloud provider instance billing")

	projectID := os.Getenv("OVH_CLOUD_PROJECT_INSTANCE_BILLING_PROJECT_ID")

	projectInstances, err := api.GetCloudProjectInstances(ovhClient, projectID)
	if err != nil {
		logger.Error().Err(err).Msgf("Failed to retrieve instances: %v", err)
	}

	var flavors []models.Flavor
	for _, instance := range projectInstances {
		if api.FindFlavorByID(flavors, instance.FlavorID) == nil {
			flavor, err := api.GetCloudProjectFlavor(ovhClient, projectID, instance.FlavorID)
			if err != nil {
				logger.Error().Err(err).Msgf("Failed to retrieve flavor: %v", err)
			} else {
				flavors = append(flavors, flavor)
			}
		}
	}

	for _, instance := range projectInstances {
		flavor := api.FindFlavorByID(flavors, instance.FlavorID)
		planType := "undefined"
		if instance.PlanCode != nil && flavor.PlanCodes.Hourly != nil && flavor.PlanCodes.Monthly != nil {
			switch {
			case *instance.PlanCode == *flavor.PlanCodes.Hourly:
				planType = "hourly"
			case *instance.PlanCode == *flavor.PlanCodes.Monthly:
				planType = "monthly"
			}
		}

		setCloudProjectInstanceBilling(projectID, instance.ID, instance.Name, planType, *instance.PlanCode, instance.MonthlyBilling.Since, instance.MonthlyBilling.Status, 1)
	}
}

func setupCacheUpdater(ovhClient *ovh.Client) {
	intervalStr := os.Getenv("OVH_CACHE_UPDATE_INTERVAL")
	intervalSeconds, err := strconv.Atoi(intervalStr)
	if err != nil {
		log.Fatalf("Failed to parse OVH_CACHE_UPDATE_INTERVAL: %v", err)
	}
	cacheUpdateInterval := time.Duration(intervalSeconds) * time.Second

	logger.Info().Msgf("setting up cache updater to sync every %v", cacheUpdateInterval)

	ticker := time.NewTicker(cacheUpdateInterval)
	defer ticker.Stop()

	for {
		updateCloudProviderInstanceBilling(ovhClient)

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
		log.Fatalf("Failed to create OVH client: %v", err)
	}

	http.HandleFunc("/ping", pingHandler)
	http.Handle("/metrics", promhttp.Handler())

	serverPort := os.Getenv("SERVER_PORT")
	formattedServerPort := fmt.Sprintf(":%s", serverPort)
	logger.Info().Msgf("server started on port %s", formattedServerPort)

	go setupCacheUpdater(ovhClient)

	return http.ListenAndServe(formattedServerPort, nil)
}
