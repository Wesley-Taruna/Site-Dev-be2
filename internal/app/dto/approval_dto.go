package dto

import "time"

// PendingApprovalResponseGet is one row of GET /v1/task/pending-approvals
// (site-dev-api PendingApprovalResp).
type PendingApprovalResponseGet struct {
	ApprovalDetailID uint      `json:"approval_detail_id"`
	TaskID           string    `json:"task_id"`
	SiteID           string    `json:"site_id"`
	SiteName1        string    `json:"site_name_1"`
	SiteAddress1     string    `json:"site_address_1"`
	SiteAddress2     string    `json:"site_address_2"`
	SiteAddress3     string    `json:"site_address_3"`
	TaskStatus       string    `json:"task_status"`
	CreatedBy        int64     `json:"created_by"`
	CreatedByName    string    `json:"created_by_name"`
	Level            int       `json:"level"`
	SubmittedAt      time.Time `json:"submitted_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// ApprovalHistoryResponseGet is one row of GET /v1/task/approval-history
// (site-dev-api ApprovalHistoryResp).
type ApprovalHistoryResponseGet struct {
	ApprovalDetailID uint       `json:"approval_detail_id"`
	TaskID           string     `json:"task_id"`
	SiteID           string     `json:"site_id"`
	SiteName1        string     `json:"site_name_1"`
	SiteAddress1     string     `json:"site_address_1"`
	SiteAddress2     string     `json:"site_address_2"`
	SiteAddress3     string     `json:"site_address_3"`
	TaskStatus       string     `json:"task_status"`
	CreatedBy        int64      `json:"created_by"`
	CreatedByName    string     `json:"created_by_name"`
	Level            int        `json:"level"`
	Status           string     `json:"status"`
	Notes            string     `json:"notes"`
	ActedAt          *time.Time `json:"acted_at"`
	CreatedAt        time.Time  `json:"created_at"`
}
