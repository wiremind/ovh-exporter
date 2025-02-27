package network

import (
	"strconv"
	"time"

	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/api"
	"github.com/wiremind/ovh-exporter/pkg/ovhsdk/models"
)

var dedicatedServerSubscription = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_dedicated_server_subscription",
		Help: "Tracks the subscription for OVH dedicated servers.",
	},
	[]string{"server_id", "status", "creation", "expiration", "engaged_up_to", "renewal_type", "renew_automatic", "renew_period", "renew_manual_payment", "renew_forced", "renew_delete_at_expiration"},
)

var dedicatedServerSubscriptionExpirationTimestamp = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ovh_exporter_dedicated_server_subscription_expiration_timestamp",
		Help: "Tracks the subscription expiration for OVH dedicated servers.",
	},
	[]string{"server_id"},
)

func setDedicatedServerSubscriptionExpirationTimestamp(server models.Server, amount float64) {
	dedicatedServerSubscriptionExpirationTimestamp.With(prometheus.Labels{
		"server_id": server.ID,
	}).Set(amount)
}

func setDedicatedServerSubscription(server models.Server, serviceinfos models.ServiceInfo, amount float64) {
	renewAutomatic := false
	if serviceinfos.Renew != nil {
		renewAutomatic = serviceinfos.Renew.Automatic
	}

	renewDeleteAtExpiration := false
	if serviceinfos.Renew != nil {
		renewDeleteAtExpiration = serviceinfos.Renew.DeleteAtExpiration
	}

	renewForced := false
	if serviceinfos.Renew != nil {
		renewForced = serviceinfos.Renew.Forced
	}

	renewManualPayment := false
	if serviceinfos.Renew != nil && serviceinfos.Renew.ManualPayment != nil {
		renewManualPayment = *serviceinfos.Renew.ManualPayment
	}

	renewPeriod := "-1"
	if serviceinfos.Renew != nil && serviceinfos.Renew.Period != nil {
		renewPeriod = strconv.Itoa(*serviceinfos.Renew.Period)
	}

	engagedUpTo := time.Unix(0, 0)
	if serviceinfos.EngagedUpTo != nil {
		engagedUpTo = *serviceinfos.EngagedUpTo
	}

	dedicatedServerSubscription.With(prometheus.Labels{
		"server_id":                  server.ID,
		"status":                     serviceinfos.Status.String(),
		"creation":                   serviceinfos.Creation.Format("2006-01-02 15:04:05"),
		"expiration":                 serviceinfos.Expiration.Format("2006-01-02 15:04:05"),
		"engaged_up_to":              engagedUpTo.Format("2006-01-02 15:04:05"),
		"renewal_type":               serviceinfos.RenewalType.String(),
		"renew_automatic":            strconv.FormatBool(renewAutomatic),
		"renew_period":               renewPeriod,
		"renew_manual_payment":       strconv.FormatBool(renewManualPayment),
		"renew_forced":               strconv.FormatBool(renewForced),
		"renew_delete_at_expiration": strconv.FormatBool(renewDeleteAtExpiration),
	}).Set(amount)
}

func updateDedicatedServerSubscription(ovhClient *ovh.Client, server models.Server) {
	logger.Info().Msgf("updating dedicated server subscription for server %s", server.ID)

	serviceInfos, err := api.GetDedicatedServerServiceInfos(ovhClient, server.ID)
	if err != nil {
		logger.Error().Msgf("failed to retrieve dedicated server serviceinfos: %v", err)
		return
	}

	setDedicatedServerSubscription(server, serviceInfos, 1)
	setDedicatedServerSubscriptionExpirationTimestamp(server, float64(serviceInfos.Expiration.Unix()))
}

func updateDedicatedServersSubscription(ovhClient *ovh.Client) {
	logger.Info().Msgf("updating dedicated server subscription")

	servers, err := api.GetDedicatedServers(ovhClient)
	if err != nil {
		logger.Error().Msgf("failed to retrieve servers: %v", err)
		return
	}

	for _, server := range servers {
		updateDedicatedServerSubscription(ovhClient, server)
	}
}
