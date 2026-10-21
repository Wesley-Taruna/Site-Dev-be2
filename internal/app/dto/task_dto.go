package dto

import "time"

// TaskRequestGet holds the query of GET /v1/task. Page comes from the
// familabs paginator; the page size is fixed at TaskPageSize (same as
// site-dev-api, which ignores per_page/limit for this list).
type TaskRequestGet struct {
	Page    int
	SiteID  string
	Status  string
	Search  string
	SortBy  string
	SortDir string
}

// ListPageSize is the fixed number of rows per page of every site-dev list
// (site-dev-api's Pagination.GetLimit: its "limit" is never filled from the
// query, so per_page/limit are ignored and 20 rows are always fetched).
const ListPageSize = 20

// TaskPageSize is the fixed number of rows per page of GET /v1/task.
const TaskPageSize = ListPageSize

// TaskResponseGet is one row of GET /v1/task. JSON names match
// site-dev-api's TaskResp exactly (no omitempty: empty values are sent).
type TaskResponseGet struct {
	TaskID       string    `json:"task_id"`
	SiteID       string    `json:"site_id"`
	SiteName1    string    `json:"site_name_1"`
	SiteAddress1 string    `json:"site_address_1"`
	SiteAddress2 string    `json:"site_address_2"`
	SiteAddress3 string    `json:"site_address_3"`
	TaskStatus   string    `json:"task_status"`
	CreatedBy    int64     `json:"created_by"`
	UpdatedBy    int64     `json:"updated_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TaskHistoryResponseGet is one row of GET /v1/task/:id/history. JSON names
// match site-dev-api's TaskHistoryResp exactly.
type TaskHistoryResponseGet struct {
	ID        uint      `json:"id"`
	Action    string    `json:"action"`
	ActorID   int64     `json:"actor_id"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

// TaskStatusCountResponseGet is one row of GET /v1/task/status/count and
// GET /v1/dashboard/task/status/count (site-dev-api TaskStatusCountResp).
type TaskStatusCountResponseGet struct {
	TaskStatus string `json:"task_status"`
	Count      int64  `json:"count"`
}

// TaskDetailApprovalItem is one approver level inside the task detail
// (site-dev-api TaskFullDetailItemApproval).
type TaskDetailApprovalItem struct {
	ID             uint       `json:"id"`
	Level          int        `json:"level"`
	ApproverRoleID string     `json:"approver_role_id"`
	ApproverID     int64      `json:"approver_id"`
	ApproverName   string     `json:"approver_name"`
	Status         string     `json:"status"`
	Notes          string     `json:"notes"`
	CreatedAt      time.Time  `json:"created_at"`
	ActedAt        *time.Time `json:"acted_at"`
}

// TaskAttachmentResponseGet is one attachment in the task detail and in
// GET /v1/task/:id/attachment (site-dev-api TaskFullDetailAttachment).
type TaskAttachmentResponseGet struct {
	ID         uint      `json:"id"`
	FileName   string    `json:"file_name"`
	FileURL    string    `json:"file_url"`
	UploadedBy int64     `json:"uploaded_by"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// TaskDetailTaskSection is the "task" part of the task detail
// (site-dev-api TaskFullDetailTaskSection).
type TaskDetailTaskSection struct {
	TaskID     string                   `json:"task_id"`
	TaskStatus string                   `json:"task_status"`
	Approval   []TaskDetailApprovalItem `json:"approval"`
	History    []TaskHistoryResponseGet `json:"history"`
}

// TaskDetailResponseGet is GET /v1/task/:id (site-dev-api
// TaskFullDetailResp): site, on-site context (null if never filled in),
// task with approval chain and history, attachments.
type TaskDetailResponseGet struct {
	Site       SiteResponseGet             `json:"site"`
	Context    *SiteContextResponseGet     `json:"context"`
	Task       TaskDetailTaskSection       `json:"task"`
	Attachment []TaskAttachmentResponseGet `json:"attachment"`
}
