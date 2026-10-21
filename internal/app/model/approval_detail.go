package model

import (
	"encoding/json"
	"time"
)

// ApprovalDetail is one approver level of an approval round (ApprovalDET).
// SecureApprovalCode (32 random characters, the e-mail link code) is never
// sent in JSON.
type ApprovalDetail struct {
	ID                 uint       `gorm:"column:ID;primaryKey" json:"id"`
	ApprovalHeaderID   uint       `gorm:"column:ApprovalHeaderID" json:"approval_header_id"`
	Level              int        `gorm:"column:Level" json:"level"`
	ApproverRoleID     string     `gorm:"column:ApproverRoleID" json:"approver_role_id"`
	ApproverID         int64      `gorm:"column:ApproverID" json:"approver_id"`
	Status             string     `gorm:"column:Status" json:"status"`
	Notes              string     `gorm:"column:Notes" json:"notes"`
	ActedAt            *time.Time `gorm:"column:ActedAt" json:"acted_at"`
	SecureApprovalCode string     `gorm:"column:SecureApprovalCode" json:"-"`
	CreatedAt          time.Time  `gorm:"column:CreatedAt" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"column:UpdatedAt" json:"updated_at"`
}

func (ApprovalDetail) TableName() string {
	return "ApprovalDET"
}

func (m *ApprovalDetail) MarshalJSON() ([]byte, error) {
	type Alias ApprovalDetail

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}
