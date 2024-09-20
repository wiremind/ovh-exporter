package models

import (
	"time"
)

type Instance struct {
	Created                     time.Time       `json:"created"`
	CurrentMonthOutgoingTraffic *int64          `json:"currentMonthOutgoingTraffic"` // Use *int64 for nullable integer
	Flavor                      Flavor          `json:"flavor"`
	ID                          string          `json:"id"`
	Image                       Image           `json:"image"`
	IpAddresses                 []IPAddress     `json:"ipAddresses"`
	MonthlyBilling              *MonthlyBilling `json:"monthlyBilling"` // Use *MonthlyBilling for nullable object
	Name                        string          `json:"name"`
	OperationIds                []string        `json:"operationIds"`
	PlanCode                    *string         `json:"planCode"` // Use *string for nullable string
	Region                      string          `json:"region"`
	RescuePassword              *string         `json:"rescuePassword"` // Use *string for nullable string
	SshKey                      *SSHKey         `json:"sshKey"`         // Use *SSHKey for nullable object
	Status                      string          `json:"status"`
}

type FlavorCapability struct {
	Enabled bool   `json:"enabled"`
	Name    string `json:"name"` // Allowed: "failoverip", "resize", "snapshot", "volume"
}

type PlanCodes struct {
	Hourly  *string `json:"hourly"`  // Use *string for nullable string
	Monthly *string `json:"monthly"` // Use *string for nullable string
}

type Image struct {
	CreationDate time.Time `json:"creationDate"`
	FlavorType   *string   `json:"flavorType"` // Use *string for nullable string
	ID           string    `json:"id"`
	MinDisk      int       `json:"minDisk"`
	MinRam       int       `json:"minRam"`
	Name         string    `json:"name"`
	PlanCode     *string   `json:"planCode"` // Use *string for nullable string
	Region       string    `json:"region"`
	Size         float64   `json:"size"` // in GiB
	Status       string    `json:"status"`
	Tags         []string  `json:"tags"`
	Type         string    `json:"type"`
	User         string    `json:"user"`
	Visibility   string    `json:"visibility"`
}

type IPAddress struct {
	GatewayIP string `json:"gatewayIp"` // Nullable, so can be empty string
	IP        string `json:"ip"`
	NetworkID string `json:"networkId"`
	Type      string `json:"type"`
	Version   int    `json:"version"`
}

type MonthlyBilling struct {
	Since  time.Time `json:"since"`
	Status string    `json:"status"` // Allowed: "activationPending", "ok"
}

type SSHKey struct {
	FingerPrint string   `json:"fingerPrint"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	PublicKey   string   `json:"publicKey"`
	Regions     []string `json:"regions"`
}

type InstanceSummary struct {
	Created                     time.Time       `json:"created"`
	CurrentMonthOutgoingTraffic *int64          `json:"currentMonthOutgoingTraffic"` // Use *int64 for nullable integer
	FlavorID                    string          `json:"flavorId"`                    // Instance flavor id
	ID                          string          `json:"id"`                          // Instance id
	ImageID                     string          `json:"imageId"`                     // Instance image id
	IpAddresses                 []IPAddress     `json:"ipAddresses"`
	MonthlyBilling              *MonthlyBilling `json:"monthlyBilling"` // Use *MonthlyBilling for nullable object
	Name                        string          `json:"name"`           // Instance name
	OperationIds                []string        `json:"operationIds"`   // Ids of pending public cloud operations
	PlanCode                    *string         `json:"planCode"`       // Use *string for nullable string
	Region                      string          `json:"region"`         // Instance region
	SshKeyID                    *string         `json:"sshKeyId"`       // Use *string for nullable string
	Status                      string          `json:"status"`         // Instance status
}
