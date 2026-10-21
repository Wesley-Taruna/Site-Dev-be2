package model

import (
	"encoding/json"
	"time"
)

type Task struct {
	TaskID     string    `gorm:"column:TaskID;primaryKey" json:"task_id"`
	SiteID     string    `gorm:"column:SiteID" json:"site_id"`
	TaskStatus string    `gorm:"column:TaskStatus" json:"task_status"`
	CreatedBy  int64     `gorm:"column:CreatedBy" json:"created_by"`
	UpdatedBy  int64     `gorm:"column:UpdatedBy" json:"updated_by"`
	CreatedAt  time.Time `gorm:"column:CreatedAt" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:UpdatedAt" json:"updated_at"`
}

func (Task) TableName() string {
	return "Tasks"
}

func (m *Task) MarshalJSON() ([]byte, error) {
	type Alias Task

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}
