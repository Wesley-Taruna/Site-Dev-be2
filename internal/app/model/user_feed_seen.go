package model

import (
	"encoding/json"
	"time"
)

// UserFeedSeen is when a user last opened a feed: the staff's Activity Log
// or the approver's "Baru" news (November scope, API-CHANGES §13.4 and
// prototype 21 Oct). New table, one row per user and feed (needs DB owner:
// plan §8).
type UserFeedSeen struct {
	UserID int64     `gorm:"column:UserID;primaryKey" json:"user_id"`
	Feed   string    `gorm:"column:Feed;primaryKey" json:"feed"`
	SeenAt time.Time `gorm:"column:SeenAt" json:"seen_at"`
}

func (UserFeedSeen) TableName() string {
	return "UserFeedSeen"
}

func (m *UserFeedSeen) MarshalJSON() ([]byte, error) {
	type Alias UserFeedSeen

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}

// ApproverNewsRow is one "Baru" item of an approver (query result, not a
// table): the task, its site and when it happened.
type ApproverNewsRow struct {
	TaskID string    `gorm:"column:TaskID"`
	SiteID string    `gorm:"column:SiteID"`
	At     time.Time `gorm:"column:At"`
}
