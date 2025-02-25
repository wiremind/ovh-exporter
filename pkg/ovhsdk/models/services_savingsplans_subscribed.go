package models

// Enum for period end actions (REACTIVATE, TERMINATE)
type PeriodEndAction string

const (
	REACTIVATE PeriodEndAction = "REACTIVATE"
	TERMINATE  PeriodEndAction = "TERMINATE"
)

// Enum for status (ACTIVE, PENDING, TERMINATED)
type Status string

const (
	ACTIVE     Status = "ACTIVE"
	PENDING    Status = "PENDING"
	TERMINATED Status = "TERMINATED"
)

type SavingsPlan struct {
	DisplayName     string          `json:"displayName"`               // Custom display name
	EndDate         string          `json:"endDate"`                   // End date of the Savings Plan
	Flavor          string          `json:"flavor"`                    // Savings Plan flavor
	ID              string          `json:"id"`                        // Unique identifier of the Savings Plan
	OfferID         string          `json:"offerId"`                   // Savings Plan commercial offer identifier
	Period          string          `json:"period"`                    // Periodicity of the Savings Plan
	PeriodEndAction PeriodEndAction `json:"periodEndAction"`           // Action when reaching the end of the period (REACTIVATE or TERMINATE)
	PeriodEndDate   string          `json:"periodEndDate"`             // End date of the current period
	PeriodStartDate string          `json:"periodStartDate"`           // Start date of the current period
	PlannedChanges  []PlannedChange `json:"plannedChanges"`            // Changes planned on the Savings Plan
	Size            int             `json:"size"`                      // Size of the Savings Plan
	StartDate       string          `json:"startDate"`                 // Start date of the Savings Plan
	Status          Status          `json:"status"`                    // Status of the Savings Plan (ACTIVE, PENDING, TERMINATED)
	TerminationDate *string         `json:"terminationDate,omitempty"` // Termination date (null if not scheduled for termination)
}

type PlannedChange struct {
	PlannedOn  string     `json:"plannedOn"`  // Date when the change will occur
	Properties Properties `json:"properties"` // Properties of the Savings Plan changing on planned date
}

type Properties struct {
	Status Status `json:"status"` // Status of the Savings Plan (ACTIVE, PENDING, TERMINATED)
}
