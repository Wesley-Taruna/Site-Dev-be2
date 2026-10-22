package model

import (
	"encoding/json"
	"time"
)

// SiteAds is the ADS data of a site, the only thing the web portal edits
// (November scope, API-CHANGES §11). AdsValue = Rupiah per day, NULL = not
// filled in. New table SiteAds, needs DB owner (plan §8); what ADS means
// exactly is still open (review §5 #11).
type SiteAds struct {
	SiteID    string    `gorm:"column:SiteID;primaryKey" json:"site_id"`
	AdsValue  *int64    `gorm:"column:AdsValue" json:"ads_value"`
	AdsNote   string    `gorm:"column:AdsNote" json:"ads_note"`
	UpdatedBy int64     `gorm:"column:UpdatedBy" json:"updated_by"`
	UpdatedAt time.Time `gorm:"column:UpdatedAt" json:"updated_at"`
}

func (SiteAds) TableName() string {
	return "SiteAds"
}

func (m *SiteAds) MarshalJSON() ([]byte, error) {
	type Alias SiteAds

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}

// TaskSubmittedAt is a task's first submit time (query result, not a
// table).
type TaskSubmittedAt struct {
	TaskID      string    `gorm:"column:TaskID"`
	SubmittedAt time.Time `gorm:"column:SubmittedAt"`
}

// TaskLevelNote is the notes of a level in a given status, by task (query
// result, not a table): the Hold reason.
type TaskLevelNote struct {
	TaskID string `gorm:"column:TaskID"`
	Notes  string `gorm:"column:Notes"`
}
