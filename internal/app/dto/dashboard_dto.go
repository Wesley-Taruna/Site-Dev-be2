package dto

import "time"

// DashboardTaskHistoryResponseGet is one row of GET
// /v1/dashboard/task/history (site-dev-api DashboardTaskHistoryResp).
type DashboardTaskHistoryResponseGet struct {
	ID         uint      `json:"id"`
	TaskID     string    `json:"task_id"`
	SiteID     string    `json:"site_id"`
	SiteName1  string    `json:"site_name_1"`
	TaskStatus string    `json:"task_status"`
	Action     string    `json:"action"`
	ActorID    int64     `json:"actor_id"`
	ActorName  string    `json:"actor_name"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
}
