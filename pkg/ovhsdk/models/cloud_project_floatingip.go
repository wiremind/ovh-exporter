package models

type FloatingIPAssociatedEntity struct {
	GatewayID *string `json:"gatewayId"`
	ID        string  `json:"id"`
	IP        string  `json:"ip"`
	Type      string  `json:"type"`
}

type FloatingIP struct {
	AssociatedEntity *FloatingIPAssociatedEntity `json:"associatedEntity"`
	ID               string                      `json:"id"`
	IP               string                      `json:"ip"`
	NetworkID        string                      `json:"networkId"`
	Status           string                      `json:"status"`
}
