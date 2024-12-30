package network

import (
	"os"
	"strings"
	"time"

	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/api"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

var cloudProjectInstanceBilling = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_cloud_project_instance_billing",
		Help: "Tracks the billing for OVH cloud project instances.",
	},
	[]string{"project_id", "instance_id", "instance_name", "billing_type", "billing_code", "billing_monthly_date", "billing_monthly_status"},
)

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

func updateCloudProviderInstanceBillingPerInstance(projectID string, instance models.InstanceSummary, flavors []models.Flavor) {
	logger.Info().Msgf("updating cloud provider instance billing for instance %s", instance.ID)

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
	instancePlanCode := "undefined"
	if instance.PlanCode != nil {
		instancePlanCode = *instance.PlanCode
	}
	instanceMonthlyBillingSince := time.Unix(0, 0)
	instanceMonthlyBillingStatus := "undefined"
	if instance.MonthlyBilling != nil {
		instanceMonthlyBillingSince = instance.MonthlyBilling.Since
		instanceMonthlyBillingStatus = instance.MonthlyBilling.Status
	}

	setCloudProjectInstanceBilling(projectID, instance.ID, instance.Name, planType, instancePlanCode, instanceMonthlyBillingSince, instanceMonthlyBillingStatus, 1)
}

func updateCloudProviderInstanceBillingPerProjectID(ovhClient *ovh.Client, projectID string) {
	logger.Info().Msgf("updating cloud provider instance billing for project %s", projectID)

	projectInstances, err := api.GetCloudProjectInstances(ovhClient, projectID)
	if err != nil {
		logger.Error().Msgf("Failed to retrieve instances: %v", err)
		return
	}

	flavors, err := api.GetCloudProjectFlavorsPerInstances(ovhClient, projectID, projectInstances)
	if err != nil {
		logger.Error().Msgf("Failed to retrieve flavors: %v", err)
		return
	}

	for _, instance := range projectInstances {
		updateCloudProviderInstanceBillingPerInstance(projectID, instance, flavors)
	}
}

func updateCloudProviderInstanceBilling(ovhClient *ovh.Client) {
	projectIDs := os.Getenv("OVH_CLOUD_PROJECT_INSTANCE_BILLING_PROJECT_IDS")

	projectIDList := strings.Split(projectIDs, ",")

	for _, projectID := range projectIDList {
		updateCloudProviderInstanceBillingPerProjectID(ovhClient, projectID)
	}
}
