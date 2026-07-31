package network

import (
	"strconv"
	"strings"

	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/api"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

// Defining the gauge vector for saving plans
var servicesSavingsPlansSubscribedPlanSize = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_services_savingsplans_subscribed_plan_size",
		Help: "Tracks size of savings plans subscribed on the service.",
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
	},
)

func setServiceSavingsPlansSubscribedPlanSize(projectID string, instanceName string, serviceID string, savingsPlanPeriod string, savingsPlanFlavor string, savingsPlanID string, savingsPlanStatus string, savingsPlanPeriodStartDate string, savingsPlanPeriodEndDate string, savingsPlanSize int) {
	servicesSavingsPlansSubscribedPlanSize.With(prometheus.Labels{
		"project_id":              projectID,
		"instance_name":           instanceName,
		"period":                  savingsPlanPeriod,
		"flavor":                  strings.ToLower(savingsPlanFlavor),
		"service_id":              serviceID,
		"savings_plan_id":         savingsPlanID,
		"savings_plan_status":     savingsPlanStatus,
		"savings_plan_start_date": savingsPlanPeriodStartDate,
		"savings_plan_end_date":   savingsPlanPeriodEndDate,
	}).Set(float64(savingsPlanSize))
}

// Function to update the savings plan subscription per service
func updateServiceSavingsPlansSubscribed(ovhClient *ovh.Client, serviceID int, projectID string) {
	logger.Info().Msgf("updating service savings plan subscription for service %d", serviceID)

	err := RefreshScope(
		ServiceScope{ServiceID: strconv.Itoa(serviceID)},
		CollectorServicesSavingsPlansSubscribed,
		func() ([]models.SavingsPlan, error) {
			return api.GetServicesSavingPlansSubscribed(ovhClient, serviceID)
		},
		func(savingsPlan models.SavingsPlan) {
			logger.Info().Msgf("processing savings plan %s for service %d", savingsPlan.ID, serviceID)
			setServiceSavingsPlansSubscribedPlanSize(
				projectID,
				savingsPlan.DisplayName,
				strconv.Itoa(serviceID),
				savingsPlan.Period,
				savingsPlan.Flavor,
				savingsPlan.ID,
				string(savingsPlan.Status),
				savingsPlan.PeriodStartDate,
				savingsPlan.PeriodEndDate,
				savingsPlan.Size,
			)
		},
		servicesSavingsPlansSubscribedPlanSize,
	)
	if err != nil {
		logger.Error().Msgf("failed to retrieve savings plans for service %d: %v", serviceID, err)
	}
}

// Function to update the savings plan subscription for all services
func updateAllServicesSavingsPlansSubscribed(ovhClient *ovh.Client) {
	// Loop through each projectID in the projectIDList
	for _, projectID := range projectIDsFromEnv("OVH_CLOUD_PROJECT_INSTANCE_BILLING_PROJECT_IDS") {

		opts := &api.Options{
			ResourceName: &projectID,
		}

		result, err := api.GetServices(ovhClient, opts)
		if err != nil {
			apiErrors.WithLabelValues(CollectorServicesSavingsPlansSubscribed).Inc()
			logger.Error().Msgf("error retrieving services for projectID %s: %v", projectID, err)
			continue
		}
		for _, service := range result {
			updateServiceSavingsPlansSubscribed(ovhClient, service, projectID)
		}
	}
}
