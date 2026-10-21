package dto

import "time"

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
