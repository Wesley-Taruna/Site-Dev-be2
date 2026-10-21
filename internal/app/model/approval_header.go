package model

import (
	"encoding/json"
	"time"
)

// ApprovalHeader is one approval round of a task (ApprovalHDR).
type ApprovalHeader struct {
	ID            uint      `gorm:"column:ID;primaryKey" json:"id"`
	TaskID        string    `gorm:"column:TaskID" json:"task_id"`
	TotalLevel    int       `gorm:"column:TotalLevel" json:"total_level"`
	CurrentLevel  int       `gorm:"column:CurrentLevel" json:"current_level"`
	OverallStatus string    `gorm:"column:OverallStatus" json:"overall_status"`
	SubmittedBy   int64     `gorm:"column:SubmittedBy" json:"submitted_by"`
	SubmittedAt   time.Time `gorm:"column:SubmittedAt" json:"submitted_at"`
	CreatedAt     time.Time `gorm:"column:CreatedAt" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:UpdatedAt" json:"updated_at"`
}

func (ApprovalHeader) TableName() string {
	return "ApprovalHDR"
}

func (m *ApprovalHeader) MarshalJSON() ([]byte, error) {
	type Alias ApprovalHeader

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}
