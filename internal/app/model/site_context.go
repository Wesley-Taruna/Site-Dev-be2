package model

import (
	"encoding/json"
	"time"
)

// SiteContext is the optional on-site survey ("Konteks On-Site") of a site,
// one row per site (primary key = SiteID).
type SiteContext struct {
	SiteID                 string    `gorm:"column:SiteID;primaryKey" json:"site_id"`
	FootTrafficObservation string    `gorm:"column:FootTrafficObservation" json:"foot_traffic_observation"`
	PeakHours              string    `gorm:"column:PeakHours" json:"peak_hours"`
	ParkingAvailability    *string   `gorm:"column:ParkingAvailability" json:"parking_availability"`
	PublicTransport        string    `gorm:"column:PublicTransport" json:"public_transport"`
	AnchorLandmark         string    `gorm:"column:AnchorLandmark" json:"anchor_landmark"`
	ObservedDemographics   string    `gorm:"column:ObservedDemographics" json:"observed_demographics"`
	CreatedBy              int64     `gorm:"column:CreatedBy" json:"created_by"`
	UpdatedBy              int64     `gorm:"column:UpdatedBy" json:"updated_by"`
	CreatedAt              time.Time `gorm:"column:CreatedAt" json:"created_at"`
	UpdatedAt              time.Time `gorm:"column:UpdatedAt" json:"updated_at"`
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
