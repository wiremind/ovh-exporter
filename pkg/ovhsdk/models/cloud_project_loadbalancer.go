package models

import (
	"time"
)

type LoadBalancer struct {
	CreatedAt          time.Time `json:"createdAt"`
	FlavorID           string    `json:"flavorId"`
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	OperatingStatus    string    `json:"operatingStatus"`
	ProvisioningStatus string    `json:"provisioningStatus"`
	UpdatedAt          time.Time `json:"updatedAt"`
	VipAddress         string    `json:"vipAddress"`
	VipNetworkID       string    `json:"vipNetworkId"`
	VipSubnetID        string    `json:"vipSubnetId"`
}
