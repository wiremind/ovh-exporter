package models

type RegionService struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Region struct {
	Name     string          `json:"name"`
	Type     string          `json:"type"`
	Status   string          `json:"status"`
	Services []RegionService `json:"services"`
}
