package constants

import "fmt"

// Task status values stored in Tasks.TaskStatus. Spelling is kept exactly
// as in site-dev-api (including "SUBMITED").
const (
	TaskStatusDraft     = "DRAFT"
	TaskStatusSubmited  = "SUBMITED"
	TaskStatusApproved  = "APPROVED"
	TaskStatusCompleted = "COMPLETED"
	TaskStatusRejected  = "REJECTED"
	TaskStatusRevision  = "REVISION"
)

// TaskStatusWaiting builds the status stored while a task waits at an
// approval level, e.g. "WAITING_DEPT_HEAD" for role "DEPT_HEAD".
func TaskStatusWaiting(roleID string) string {
	return fmt.Sprintf("WAITING_%s", roleID)
}

// Task history actions stored in TaskHistories.Action.
const (
	TaskActionCreate   = "CREATE"
	TaskActionUpdate   = "UPDATE"
	TaskActionSubmit   = "SUBMIT"
	TaskActionApprove  = "APPROVE"
	TaskActionReject   = "REJECT"
	TaskActionRevision = "REVISION"
	TaskActionForward  = "FORWARD"
	TaskActionComplete = "COMPLETE"
)

// Role IDs stored in Users.RoleID / Roles.RoleID. RoleUser (site staff) is
// the only role whose lists are limited to its own records.
const (
	RoleUser     = "USER"
	RoleAdmin    = "ADMIN"
	RoleDeptHead = "DEPT_HEAD"
	RoleDivHead  = "DIV_HEAD"
)

// ApprovalDET.Status values.
const (
	ApprovalDetailStatusWaiting  = "WAITING"
	ApprovalDetailStatusPending  = "PENDING"
	ApprovalDetailStatusApproved = "APPROVED"
	ApprovalDetailStatusRejected = "REJECTED"
	ApprovalDetailStatusRevision = "REVISION"
)

// ApprovalHDR.OverallStatus values.
const (
	ApprovalOverallStatusPending  = "PENDING"
	ApprovalOverallStatusApproved = "APPROVED"
	ApprovalOverallStatusRejected = "REJECTED"
	ApprovalOverallStatusRevision = "REVISION"
)

// SiteContexts.ParkingAvailability values.
const (
	ParkingAvailable = "AVAILABLE"
	ParkingLimited   = "LIMITED"
	ParkingNone      = "NONE"
)

// ID prefixes: Sites.SiteID = "ST" + YYYYMMDD + 6 digits, Tasks.TaskID =
// "TS" + YYYYMMDD + 6 digits.
const (
	SiteIDPrefix = "ST"
	TaskIDPrefix = "TS"
)

// SiteListRequestStatuses is every task status shown on the "Site Request"
// table (GET /v1/task/site-list): all except APPROVED.
var SiteListRequestStatuses = []string{
	TaskStatusDraft,
	TaskStatusSubmited,
	TaskStatusWaiting(RoleDeptHead),
	TaskStatusWaiting(RoleDivHead),
	TaskStatusRejected,
	TaskStatusRevision,
	TaskStatusCompleted,
}
