package models

import (
	"time"
)

type Volume struct {
	AttachedTo   []string  `json:"attachedTo"`
	Bootable     bool      `json:"bootable"`
	CreationDate time.Time `json:"creationDate"`
	Description  string    `json:"description"`
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	PlanCode     *string   `json:"planCode"`
	Region       string    `json:"region"`
	Size         int       `json:"size"`
	Status       string    `json:"status"`
	Type         string    `json:"type"`
}
