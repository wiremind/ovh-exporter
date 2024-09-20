package models

type Flavor struct {
	Available         bool               `json:"available"`
	Capabilities      []FlavorCapability `json:"capabilities"`
	Disk              int                `json:"disk"`
	ID                string             `json:"id"`
	InboundBandwidth  *int64             `json:"inboundBandwidth"` // Use *int64 for nullable integer
	Name              string             `json:"name"`
	OsType            string             `json:"osType"`
	OutboundBandwidth *int64             `json:"outboundBandwidth"` // Use *int64 for nullable integer
	PlanCodes         PlanCodes          `json:"planCodes"`
	Quota             int                `json:"quota"`
	RAM               int                `json:"ram"`
	Region            string             `json:"region"`
	Type              string             `json:"type"`
	VCPUs             int                `json:"vcpus"`
}
