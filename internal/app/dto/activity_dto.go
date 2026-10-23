package dto

import "time"

// ActivityRequestGet is the query of GET /v1/activity: filter all (default)
// | action | mine, page, limit (default 20, max 100). mine = staff: the
// approvers' moves on my requests; Dept / Div Head: my own decisions.
type ActivityRequestGet struct {
	Filter string
	Offset int
	Limit  int
}

// ActivityResponseGet is one Activity Log row (November scope,
// API-CHANGES §13.4, prototype ActivityResp). task_status is the request's
// status now. No note or reason on any row (Wes, 21 + 23 Oct; it is in the
// request's history). notice: DIV_HEAD_RETURNED for the Dept Head ("Untuk
// Anda"), left out otherwise (staff rows never have it).
type ActivityResponseGet struct {
	ID          uint      `json:"id"`
	TaskID      string    `json:"task_id"`
	SiteID      string    `json:"site_id"`
	SiteName1   string    `json:"site_name_1"`
	TaskStatus  string    `json:"task_status"`
	Action      string    `json:"action"`
	ActorID     int64     `json:"actor_id"`
	ActorName   string    `json:"actor_name"`
	ActorRole   string    `json:"actor_role"`
	CreatedAt   time.Time `json:"created_at"`
	Round       int       `json:"round"`
	NeedsAction bool      `json:"needs_action"`
	Unread      bool      `json:"unread"`
	Notice      string    `json:"notice,omitempty"`
}

// UnreadCountResponse is GET /v1/activity/unread-count.
type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

// ApproverNewsResponseGet is one row of GET /v1/task/approver-news
// (prototype ApproverNewsResp, 21 Oct): kind NEW_TASK | DIV_HEAD_RETURNED.
type ApproverNewsResponseGet struct {
	TaskID    string    `json:"task_id"`
	SiteID    string    `json:"site_id"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
}
