package dto

import (
	"encoding/json"
	"time"

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
	return o
}

// Messages implements validation.Request.
func (r SiteContextRequestUpsert) Messages() map[string]string {
	return map[string]string{}
}
