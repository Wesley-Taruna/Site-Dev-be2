package model

import (
	"encoding/json"
	"time"
)

// SystemSetting is one admin-editable key/value (today: the daily AI insight
// and geocoding limits).
type SystemSetting struct {
	ID           uint      `gorm:"column:ID;primaryKey" json:"id"`
	SettingKey   string    `gorm:"column:SettingKey" json:"setting_key"`
	SettingValue string    `gorm:"column:SettingValue" json:"setting_value"`
	UpdatedBy    int64     `gorm:"column:UpdatedBy" json:"updated_by"`
	CreatedAt    time.Time `gorm:"column:CreatedAt" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:UpdatedAt" json:"updated_at"`
}

func (SystemSetting) TableName() string {
	return "SystemSettings"
}

func (m *SystemSetting) MarshalJSON() ([]byte, error) {
	type Alias SystemSetting

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}
