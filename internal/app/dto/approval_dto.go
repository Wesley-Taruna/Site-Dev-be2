package dto

import (
	"time"
	"unicode/utf8"

	"gitlab.com/fami-app-v2/familib/familabs/validation"
)

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
	// November scope (API-CHANGES §13.3): submit round, > 1 = "Pernah
	// dikirim sebelumnya"
	SubmitCount int `json:"submit_count,omitempty"`
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
	// November scope (prototype 21 Oct): for a task now REVISION/REJECTED,
	// the role that returned it
	RejectedByRole string `json:"rejected_by_role,omitempty"`
}

// TaskActionRequest is the body of POST /v1/task/:id/approve: decision
// APPROVE, REJECT ("Tolak final") or REVISION ("Tolak, bisa direvisi"),
// and since the November scope HOLD and RELEASE (API-CHANGES §13.1).
// Notes (catatan) at most 500 characters; required only for HOLD (Wes, 26
// Oct: optional for REJECT and REVISION, site-dev-api required them).
type TaskActionRequest struct {
	Decision string `json:"decision" valid:"required|in:APPROVE,REJECT,REVISION,HOLD,RELEASE"`
	Notes    string `json:"notes"`
}

// Validate implements validation.Request (site-dev-api: max=500).
func (r TaskActionRequest) Validate() *validation.Output {
	o := &validation.Output{Valid: true}
	if r.Decision == "HOLD" && r.Notes == "" {
		o.Failure("notes.required", "Alasan Hold Status wajib diisi.")
	}
	if len(r.Notes) > 500 {
		o.Failure("notes.lte", "The notes may not be greater than 500")
	}
	return o
}

// Messages implements validation.Request.
func (r TaskActionRequest) Messages() map[string]string {
	return map[string]string{}
}

// TaskCancelRequest is the body of POST /v1/task/:id/cancel (Wes, 26 Oct):
// the staff cancel a request returned to them; notes optional, at most 500
// characters. An empty body is fine.
type TaskCancelRequest struct {
	Notes string `json:"notes"`
}

// Validate implements validation.Request.
func (r TaskCancelRequest) Validate() *validation.Output {
	o := &validation.Output{Valid: true}
	if utf8.RuneCountInString(r.Notes) > 500 {
		o.Failure("notes.lte", "Catatan maksimal 500 karakter.")
	}
	return o
}

// Messages implements validation.Request.
func (r TaskCancelRequest) Messages() map[string]string {
	return map[string]string{}
}

// ApprovalEmailTaskPayload is the queue job of POST
// /v1/queue/approval-email (site-dev-api dto, unchanged).
type ApprovalEmailTaskPayload struct {
	To           string `json:"to"`
	ToName       string `json:"to_name"`
	TaskID       string `json:"task_id"`
	SiteName     string `json:"site_name"`
	SubmittedBy  string `json:"submitted_by"`
	ApproverRole string `json:"approver_role"`
	Notes        string `json:"notes"`
	ApprovalCode string `json:"approval_code"`
}
