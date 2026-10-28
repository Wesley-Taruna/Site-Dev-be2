package dto

import "time"

// TaskRequestGet holds the query of GET /v1/task. Page comes from the
// familabs paginator. Since Wes, 28 Oct (simple app, API-CHANGES-SIMPLE
// §2.4) the page size comes from limit / per_page (default TaskPageSize,
// max 100); site-dev-api ignored them and always sent 20.
type TaskRequestGet struct {
	Page int
	// Limit: rows per page; 0 = TaskPageSize.
	Limit  int
	SiteID string
	Status string
	// StatusNot (November scope, review §5 #7, OPEN): statuses to leave
	// out, comma-separated (e.g. "DRAFT" = "all except drafts" for the
	// approvers' Semua tab). Used only when Status is empty; empty = no
	// filter, so today's calls answer as before.
	StatusNot []string
	Search    string
	SortBy    string
	SortDir   string
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
	// November scope (API-CHANGES §13.3, prototype TaskResp): submit round
	// (> 1 = "Pernah dikirim sebelumnya") and, for REVISION/REJECTED, the
	// role that returned it. omitempty: answers on old data stay the same.
	SubmitCount    int    `json:"submit_count,omitempty"`
	RejectedByRole string `json:"rejected_by_role,omitempty"`
}

// TaskHistoryResponseGet is one row of GET /v1/task/:id/history. JSON names
// match site-dev-api's TaskHistoryResp exactly.
type TaskHistoryResponseGet struct {
	ID        uint      `json:"id"`
	Action    string    `json:"action"`
	ActorID   int64     `json:"actor_id"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	// November scope (API-CHANGES §13.3): role at that moment and submit
	// round; left out on rows written before the change (NULL).
	ActorRole string `json:"actor_role,omitempty"`
	Round     int    `json:"round,omitempty"`
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
	// November scope, as TaskResponseGet
	SubmitCount    int    `json:"submit_count,omitempty"`
	RejectedByRole string `json:"rejected_by_role,omitempty"`
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
