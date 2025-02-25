package network

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/api"
)

// Defining the gauge vector for saving plans
var servicesSavingsPlansSubscribed = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_services_savingsplans_subscribed",
		Help: "Tracks if a saving plan is subscribed on the service.",
	},
	[]string{
		"project_id",
		"instance_name",
		"period",
		"flavor",
		"service_id",
		"savings_plan_id",
		"savings_plan_status",
		"savings_plan_start_date",
		"savings_plan_end_date",
		"savings_plan_size",
	},
)

// Function to set the value in Prometheus gauge
func setServiceSavingsPlansSubscribed(projectID string, instanceName string, serviceID string, savingsPlanPeriod string, savingsPlanFlavor string, savingsPlanID string, savingsPlanStatus string, savingsPlanPeriodStartDate string, savingsPlanPeriodEndDate string, savingsPlanSize string) {
	servicesSavingsPlansSubscribed.With(prometheus.Labels{
		"project_id":              projectID,
		"instance_name":           instanceName,
		"period":                  savingsPlanPeriod,
		"flavor":                  savingsPlanFlavor,
		"service_id":              serviceID,
		"savings_plan_id":         savingsPlanID,
		"savings_plan_status":     savingsPlanStatus,
		"savings_plan_start_date": savingsPlanPeriodStartDate,
		"savings_plan_end_date":   savingsPlanPeriodEndDate,
		"savings_plan_size":       savingsPlanSize,
	}).Set(1)
}

// Function to update the savings plan subscription per service
func updateServiceSavingsPlansSubscribed(ovhClient *ovh.Client, serviceID int, projectID string) {
	logger.Info().Msgf("updating service savings plan subscription for service %d", serviceID)

	// Retrieve the savings plans for the given service
	savingsPlans, err := api.GetServicesSavingPlansSubscribed(ovhClient, serviceID)
	if err != nil {
		logger.Error().Msgf("failed to retrieve savings plans for service %d: %v", serviceID, err)
		return
	}

	// Check if savingsPlans slice is empty
	if len(savingsPlans) == 0 {
		logger.Warn().Msgf("no savings plans found for service %d", serviceID)
		return
	}

	// Iterate through all savings plans (in case there are multiple)
	for _, savingsPlan := range savingsPlans {
		// Log each plan being processed
		logger.Info().Msgf("processing savings plan %s for service %d", savingsPlan.ID, serviceID)

		// Set the values in the Prometheus gauge for each plan
		setServiceSavingsPlansSubscribed(
			projectID,
			savingsPlan.DisplayName,
			strconv.Itoa(serviceID),
			savingsPlan.Period,
			savingsPlan.Flavor,
			savingsPlan.ID,
			string(savingsPlan.Status),
			savingsPlan.PeriodStartDate,
			savingsPlan.PeriodEndDate,
			strconv.Itoa(savingsPlan.Size),
		)
	}
}

// Function to update the savings plan subscription for all services
func updateAllServicesSavingsPlansSubscribed(ovhClient *ovh.Client) {
	projectIDs := os.Getenv("OVH_CLOUD_PROJECT_INSTANCE_BILLING_PROJECT_IDS")

	projectIDList := strings.Split(projectIDs, ",")

	// Loop through each projectID in the projectIDList
	for _, projectID := range projectIDList {

		opts := &api.Options{
			ResourceName: &projectID,
		}

		result, err := api.GetServices(ovhClient, opts)
		if err != nil {

			log.Printf("error retrieving services for projectID %s: %v", projectID, err)
			continue
		}
		for _, service := range result {
			updateServiceSavingsPlansSubscribed(ovhClient, service, projectID)
		}
	}
}
