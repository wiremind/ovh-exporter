package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type RenewalType int

const (
	AutomaticForcedProduct RenewalType = iota
	AutomaticV2012
	AutomaticV2014
	AutomaticV2016
	AutomaticV2024
	Manual
	OneShot
	Option
)

func (r RenewalType) String() string {
	return [...]string{
		"automaticForcedProduct",
		"automaticV2012",
		"automaticV2014",
		"automaticV2016",
		"automaticV2024",
		"manual",
		"oneShot",
		"option",
	}[r]
}

func (r RenewalType) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func (r *RenewalType) UnmarshalJSON(data []byte) error {
	var renewalType string
	if err := json.Unmarshal(data, &renewalType); err != nil {
		return err
	}

	switch renewalType {
	case "automaticForcedProduct":
		*r = AutomaticForcedProduct
	case "automaticV2012":
		*r = AutomaticV2012
	case "automaticV2014":
		*r = AutomaticV2014
	case "automaticV2016":
		*r = AutomaticV2016
	case "automaticV2024":
		*r = AutomaticV2024
	case "manual":
		*r = Manual
	case "oneShot":
		*r = OneShot
	case "option":
		*r = Option
	default:
		return fmt.Errorf("invalid RenewalType: %s", renewalType)
	}
	return nil
}

type ServiceStatus int

const (
	AutoRenewInProgress ServiceStatus = iota
	Expired
	InCreation
	OK
	PendingDebt
	UnPaid
)

func (s ServiceStatus) String() string {
	return [...]string{
		"autorenewInProgress",
		"expired",
		"inCreation",
		"ok",
		"pendingDebt",
		"unPaid",
	}[s]
}

func (s ServiceStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *ServiceStatus) UnmarshalJSON(data []byte) error {
	var status string
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}

	switch status {
	case "autorenewInProgress":
		*s = AutoRenewInProgress
	case "expired":
		*s = Expired
	case "inCreation":
		*s = InCreation
	case "ok":
		*s = OK
	case "pendingDebt":
		*s = PendingDebt
	case "unPaid":
		*s = UnPaid
	default:
		return fmt.Errorf("invalid ServiceStatus: %s", status)
	}
	return nil
}

type Renew struct {
	Automatic          bool        `json:"automatic"`
	DeleteAtExpiration bool        `json:"deleteAtExpiration"`
	Forced             bool        `json:"forced"`
	ManualPayment      *bool       `json:"manualPayment,omitempty"` // Nullable
	Period             *int        `json:"period,omitempty"`        // Nullable
	RenewalType        RenewalType `json:"renewalType"`
}

type ServiceInfo struct {
	CanDeleteAtExpiration bool          `json:"canDeleteAtExpiration"`
	ContactAdmin          string        `json:"contactAdmin"`
	ContactBilling        string        `json:"contactBilling"`
	ContactTech           string        `json:"contactTech"`
	Creation              time.Time     `json:"creation"`
	Domain                string        `json:"domain"`
	EngagedUpTo           *time.Time    `json:"engagedUpTo,omitempty"` // Nullable
	Expiration            time.Time     `json:"expiration"`
	PossibleRenewPeriod   []int         `json:"possibleRenewPeriod"`
	Renew                 *Renew        `json:"renew,omitempty"` // Nullable
	RenewalType           RenewalType   `json:"renewalType"`
	ServiceId             int           `json:"serviceId"`
	Status                ServiceStatus `json:"status"`
}

func parseDate(dateStr string) (time.Time, error) {
	if len(dateStr) == 10 {
		dateStr = dateStr + "T00:00:00Z"
	}

	layout := "2006-01-02T15:04:05Z"
	parsedTime, err := time.Parse(layout, dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to parse creation date: %s", dateStr)
	}

	return parsedTime, nil
}

func (s *ServiceInfo) UnmarshalJSON(data []byte) error {
	type Alias ServiceInfo
	aux := &struct {
		Creation    string  `json:"creation"`
		EngagedUpTo *string `json:"engagedUpTo,omitempty"` // Nullable
		Expiration  string  `json:"expiration"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	s.Creation, err = parseDate(aux.Creation)
	if err != nil {
		return fmt.Errorf("unable to parse creation date: %s", aux.Creation)
	}

	if aux.EngagedUpTo != nil {
		engagedUpToTime, err := parseDate(*aux.EngagedUpTo)
		if err != nil {
			return fmt.Errorf("unable to parse engagedUpTo date: %s", *aux.EngagedUpTo)
		}
		s.EngagedUpTo = &engagedUpToTime
	}

	s.Expiration, err = parseDate(aux.Expiration)
	if err != nil {
		return fmt.Errorf("unable to parse expiration date: %s", aux.Expiration)
	}
	return nil
}
