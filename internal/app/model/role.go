package model

import "encoding/json"

// Role is one row of the Roles master table (read only here). RoleLevel
// orders the approval chain.
type Role struct {
	RoleID    string `gorm:"column:RoleID;primaryKey" json:"role_id"`
	RoleName  string `gorm:"column:RoleName" json:"role_name"`
	RoleLevel int    `gorm:"column:RoleLevel" json:"role_level"`
}

func (Role) TableName() string {
	return "Roles"
}

func (m *Role) MarshalJSON() ([]byte, error) {
	type Alias Role

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}
