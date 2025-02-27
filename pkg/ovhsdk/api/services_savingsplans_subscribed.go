package api

import (
	"strconv"

	"github.com/ovh/go-ovh/ovh"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

func GetServicesSavingPlansSubscribed(client *ovh.Client, serviceId int) ([]models.SavingsPlan, error) {
	var savingsPlans []models.SavingsPlan

	endpoint := "/services/" + strconv.Itoa(serviceId) + "/savingsPlans/subscribed"

	err := client.Get(endpoint, &savingsPlans)
	if err != nil {
		return savingsPlans, err
	}

	return savingsPlans, nil
}
