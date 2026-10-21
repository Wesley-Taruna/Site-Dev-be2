package model

import (
	"encoding/json"
	"time"
)

type TaskHistory struct {
	ID        uint      `gorm:"column:ID;primaryKey" json:"id"`
	TaskID    string    `gorm:"column:TaskID" json:"task_id"`
	Action    string    `gorm:"column:Action" json:"action"`
	ActorID   int64     `gorm:"column:ActorID" json:"actor_id"`
	Notes     string    `gorm:"column:Notes" json:"notes"`
	CreatedAt time.Time `gorm:"column:CreatedAt" json:"created_at"`
	// ActorRole is the actor's role at that moment and Round the submit
	// round the row belongs to (November scope, API-CHANGES §13.3; new
	// columns, NULL on rows written before, needs DB owner: plan §8).
	ActorRole *string `gorm:"column:ActorRole" json:"actor_role"`
	Round     *int    `gorm:"column:Round" json:"round"`
}

func (TaskHistory) TableName() string {
	return "TaskHistories"
}

func (m *TaskHistory) MarshalJSON() ([]byte, error) {
	type Alias TaskHistory

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}
