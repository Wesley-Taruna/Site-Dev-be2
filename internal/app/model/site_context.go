package model

import (
	"encoding/json"
	"time"
)

// SiteContext is the optional on-site survey ("Konteks On-Site") of a site,
// one row per site (primary key = SiteID).
type SiteContext struct {
	SiteID                 string  `gorm:"column:SiteID;primaryKey" json:"site_id"`
	FootTrafficObservation string  `gorm:"column:FootTrafficObservation" json:"foot_traffic_observation"`
	PeakHours              string  `gorm:"column:PeakHours" json:"peak_hours"`
	ParkingAvailability    *string `gorm:"column:ParkingAvailability" json:"parking_availability"`
	PublicTransport        string  `gorm:"column:PublicTransport" json:"public_transport"`
	AnchorLandmark         string  `gorm:"column:AnchorLandmark" json:"anchor_landmark"`
	ObservedDemographics   string  `gorm:"column:ObservedDemographics" json:"observed_demographics"`
	// November scope (API-CHANGES §13.5, §6 note 19 Oct, prototype
	// ecdb9be): vehicle traffic, parking note, staff competitor note. New
	// columns, NULL when not filled in (needs DB owner: plan §8).
	VehicleTrafficLevel   *string   `gorm:"column:VehicleTrafficLevel" json:"vehicle_traffic_level"`
	VehicleTrafficCount   *int      `gorm:"column:VehicleTrafficCount" json:"vehicle_traffic_count"`
	VehicleTrafficMinutes *int      `gorm:"column:VehicleTrafficMinutes" json:"vehicle_traffic_minutes"`
	VehicleTrafficNote    *string   `gorm:"column:VehicleTrafficNote" json:"vehicle_traffic_note"`
	ParkingNote           *string   `gorm:"column:ParkingNote" json:"parking_note"`
	CompetitorNote        *string   `gorm:"column:CompetitorNote" json:"competitor_note"`
	CreatedBy             int64     `gorm:"column:CreatedBy" json:"created_by"`
	UpdatedBy             int64     `gorm:"column:UpdatedBy" json:"updated_by"`
	CreatedAt             time.Time `gorm:"column:CreatedAt" json:"created_at"`
	UpdatedAt             time.Time `gorm:"column:UpdatedAt" json:"updated_at"`
}

func (SiteContext) TableName() string {
	return "SiteContexts"
}

func (m *SiteContext) MarshalJSON() ([]byte, error) {
	type Alias SiteContext

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}
