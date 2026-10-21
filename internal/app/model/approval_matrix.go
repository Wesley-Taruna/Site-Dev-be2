package model

import (
	"encoding/json"
	"time"
)

// ApprovalMatrix says which user currently approves for a role. The order
// of approval comes from Roles.RoleLevel, not from this table.
type ApprovalMatrix struct {
	ID        uint      `gorm:"column:ID;primaryKey" json:"id"`
	RoleID    string    `gorm:"column:RoleID" json:"role_id"`
	UserID    int64     `gorm:"column:UserID" json:"user_id"`
	IsActive  bool      `gorm:"column:IsActive" json:"is_active"`
	CreatedAt time.Time `gorm:"column:CreatedAt" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:UpdatedAt" json:"updated_at"`
}

func (ApprovalMatrix) TableName() string {
	return "ApprovalMatrix"
}

func (m *ApprovalMatrix) MarshalJSON() ([]byte, error) {
	type Alias ApprovalMatrix

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}

// ApprovalChainStep is one active ApprovalMatrix row with its role level
// (query result, not a table).
type ApprovalChainStep struct {
	RoleID    string
	RoleLevel int
	UserID    int64
}
