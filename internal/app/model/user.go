package model

import (
	"encoding/json"
	"time"
)

// User is one row of the Users master table. UserID holds the employee's
// NIK (numeric), the same value used in every CreatedBy/ActorID column.
type User struct {
	UserID    int64     `gorm:"column:UserID;primaryKey" json:"user_id"`
	Username  string    `gorm:"column:Username" json:"username"`
	Email     string    `gorm:"column:Email" json:"email"`
	RoleID    string    `gorm:"column:RoleID" json:"role_id"`
	CreatedAt time.Time `gorm:"column:CreatedAt" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:UpdatedAt" json:"updated_at"`
}

func (User) TableName() string {
	return "Users"
}

func (m *User) MarshalJSON() ([]byte, error) {
	type Alias User

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}
