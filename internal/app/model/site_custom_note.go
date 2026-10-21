package model

import (
	"encoding/json"
	"time"
)

// SiteCustomNote is one free note of a site survey ("Catatan Custom").
type SiteCustomNote struct {
	ID        uint      `gorm:"column:ID;primaryKey" json:"id"`
	SiteID    string    `gorm:"column:SiteID" json:"site_id"`
	Title     string    `gorm:"column:Title" json:"title"`
	Content   string    `gorm:"column:Content" json:"content"`
	CreatedAt time.Time `gorm:"column:CreatedAt" json:"created_at"`
}

func (SiteCustomNote) TableName() string {
	return "SiteCustomNotes"
}

func (m *SiteCustomNote) MarshalJSON() ([]byte, error) {
	type Alias SiteCustomNote

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}
