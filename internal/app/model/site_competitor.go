package model

import (
	"encoding/json"
	"time"
)

// SiteCompetitor is one competitor seen near a site ("Kompetitor Teramati").
type SiteCompetitor struct {
	ID             uint      `gorm:"column:ID;primaryKey" json:"id"`
	SiteID         string    `gorm:"column:SiteID" json:"site_id"`
	CompetitorName string    `gorm:"column:CompetitorName" json:"competitor_name"`
	CompetitorType string    `gorm:"column:CompetitorType" json:"competitor_type"`
	Distance       string    `gorm:"column:Distance" json:"distance"`
	CreatedAt      time.Time `gorm:"column:CreatedAt" json:"created_at"`
}

func (SiteCompetitor) TableName() string {
	return "SiteCompetitors"
}

func (m *SiteCompetitor) MarshalJSON() ([]byte, error) {
	type Alias SiteCompetitor

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}
