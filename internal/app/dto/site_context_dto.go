package dto

import (
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"

	"site-dev-backend/internal/app/constants"

	"gitlab.com/fami-app-v2/familib/familabs/validation"
)

// CompetitorResponseGet / CustomNoteResponseGet / SiteContextResponseGet
// match site-dev-api's CompetitorResp / CustomNoteResp / SiteContextResp.
type CompetitorResponseGet struct {
	ID             uint   `json:"id"`
	CompetitorName string `json:"competitor_name"`
	CompetitorType string `json:"competitor_type"`
	Distance       string `json:"distance"`
}

type CustomNoteResponseGet struct {
	ID      uint   `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type SiteContextResponseGet struct {
	SiteID                 string                  `json:"site_id"`
	FootTrafficObservation string                  `json:"foot_traffic_observation"`
	PeakHours              string                  `json:"peak_hours"`
	ParkingAvailability    string                  `json:"parking_availability"`
	PublicTransport        string                  `json:"public_transport"`
	AnchorLandmark         string                  `json:"anchor_landmark"`
	ObservedDemographics   string                  `json:"observed_demographics"`
	Competitors            []CompetitorResponseGet `json:"competitors"`
	CustomNotes            []CustomNoteResponseGet `json:"custom_notes"`
	CreatedAt              time.Time               `json:"created_at"`
	UpdatedAt              time.Time               `json:"updated_at"`
	// November scope (API-CHANGES §13.5, prototype SiteContext): left out
	// when not filled in, so answers on old data stay the same.
	VehicleTrafficLevel   string `json:"vehicle_traffic_level,omitempty"`
	VehicleTrafficCount   *int   `json:"vehicle_traffic_count,omitempty"`
	VehicleTrafficMinutes *int   `json:"vehicle_traffic_minutes,omitempty"`
	VehicleTrafficNote    string `json:"vehicle_traffic_note,omitempty"`
	ParkingNote           string `json:"parking_note,omitempty"`
	CompetitorNote        string `json:"competitor_note,omitempty"`
}

// CompetitorItem / CustomNoteItem are the lists of the survey form.
type CompetitorItem struct {
	CompetitorName string `json:"competitor_name" valid:"required"`
	CompetitorType string `json:"competitor_type"`
	Distance       string `json:"distance"`
}

type CustomNoteItem struct {
	Title   string `json:"title" valid:"required"`
	Content string `json:"content"`
}

// SiteContextRequestUpsert saves the whole "Konteks On-Site" form at once:
// the fields plus both lists, replacing what was there (site-dev-api
// SiteContextUpsertReq).
type SiteContextRequestUpsert struct {
	FootTrafficObservation string           `json:"foot_traffic_observation"`
	PeakHours              string           `json:"peak_hours"`
	ParkingAvailability    *string          `json:"parking_availability"`
	PublicTransport        string           `json:"public_transport"`
	AnchorLandmark         string           `json:"anchor_landmark"`
	ObservedDemographics   string           `json:"observed_demographics"`
	Competitors            []CompetitorItem `json:"competitors"`
	CustomNotes            []CustomNoteItem `json:"custom_notes"`
	// November scope (API-CHANGES §13.5): vehicle traffic (optional, also
	// on SUBMITED: Wes 26 Oct, only name, map point and sales area are
	// needed to send), count of vehicles in minutes, notes. No observation
	// time (dropped 21 Oct).
	VehicleTrafficLevel   string `json:"vehicle_traffic_level"`
	VehicleTrafficCount   *int   `json:"vehicle_traffic_count"`
	VehicleTrafficMinutes *int   `json:"vehicle_traffic_minutes"`
	VehicleTrafficNote    string `json:"vehicle_traffic_note"`
	ParkingNote           string `json:"parking_note"`
	CompetitorNote        string `json:"competitor_note"`
}

// UnmarshalJSON treats parking_availability "" like a missing value (the
// app sends "" for an unfilled optional field).
func (r *SiteContextRequestUpsert) UnmarshalJSON(data []byte) error {
	type alias SiteContextRequestUpsert
	if err := json.Unmarshal(data, (*alias)(r)); err != nil {
		return err
	}
	if r.ParkingAvailability != nil && *r.ParkingAvailability == "" {
		r.ParkingAvailability = nil
	}
	return nil
}

// Validate implements validation.Request: parking_availability, when
// given, is AVAILABLE, LIMITED or NONE.
func (r SiteContextRequestUpsert) Validate() *validation.Output {
	o := &validation.Output{Valid: true}
	if p := r.ParkingAvailability; p != nil && *p != "AVAILABLE" && *p != "LIMITED" && *p != "NONE" {
		o.Failure("parking_availability.in", "The selected parking availability is invalid.")
	}
	r.validateNovember(o)
	return o
}

// validateNovember checks the November survey fields (API-CHANGES §13.5;
// limits as the prototype's form).
func (r SiteContextRequestUpsert) validateNovember(o *validation.Output) {
	switch r.VehicleTrafficLevel {
	case "", constants.VehicleTrafficQuiet, constants.VehicleTrafficModerate, constants.VehicleTrafficBusy, constants.VehicleTrafficVeryBusy:
	default:
		o.Failure("vehicle_traffic_level.in", "The selected vehicle traffic level is invalid.")
	}
	if c := r.VehicleTrafficCount; c != nil {
		if *c < 0 || *c > constants.VehicleTrafficCountMax {
			o.Failure("vehicle_traffic_count.between", fmt.Sprintf("The vehicle traffic count must be between 0 and %d.", constants.VehicleTrafficCountMax))
		}
		if r.VehicleTrafficMinutes == nil {
			o.Failure("vehicle_traffic_minutes.required", "The vehicle traffic minutes field is required when vehicle traffic count is present.")
		}
	}
	if m := r.VehicleTrafficMinutes; m != nil && (*m < constants.VehicleTrafficMinutesMin || *m > constants.VehicleTrafficMinutesMax) {
		o.Failure("vehicle_traffic_minutes.between", fmt.Sprintf("The vehicle traffic minutes must be between %d and %d.", constants.VehicleTrafficMinutesMin, constants.VehicleTrafficMinutesMax))
	}
	maxLen(o, "vehicle_traffic_note", r.VehicleTrafficNote, constants.VehicleTrafficNoteMax)
	maxLen(o, "parking_note", r.ParkingNote, constants.ParkingNoteMax)
	maxLen(o, "competitor_note", r.CompetitorNote, constants.CompetitorNoteMax)
}

// maxLen fails field when value is longer than max characters.
func maxLen(o *validation.Output, field string, value string, max int) {
	if utf8.RuneCountInString(value) > max {
		o.Failure(field+".lte", fmt.Sprintf("The %s may not be greater than %d characters.", field, max))
	}
}

// Messages implements validation.Request.
func (r SiteContextRequestUpsert) Messages() map[string]string {
	return map[string]string{}
}
