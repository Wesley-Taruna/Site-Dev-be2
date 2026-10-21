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

// TaskStatusHold builds the status of a task an approver holds ("Hold
// Status", November scope, API-CHANGES §13.1), e.g. "HOLD_DEPT_HEAD".
func TaskStatusHold(roleID string) string {
	return fmt.Sprintf("HOLD_%s", roleID)
}

// TaskStatusHoldGroup is the list filter value for both HOLD_* statuses
// (GET /v1/task?status=HOLD, API-CHANGES §13.3).
const TaskStatusHoldGroup = "HOLD"

// HoldStatuses are the HOLD_* statuses of the fixed chain.
var HoldStatuses = []string{TaskStatusHold(RoleDeptHead), TaskStatusHold(RoleDivHead)}

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
	// November scope (API-CHANGES §13.1): an approver holds a request, and
	// releases it again
	TaskActionHold    = "HOLD"
	TaskActionRelease = "RELEASE"
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
	// ApprovalDetailStatusHold: the approver holds the level (notes = reason)
	ApprovalDetailStatusHold = "HOLD"
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
	// November scope (API-CHANGES §13.3): held requests are in flight too
	TaskStatusHold(RoleDeptHead),
	TaskStatusHold(RoleDivHead),
}

// Daily quotas (per user, per day in Asia/Jakarta). The limit is read from
// SystemSettings (key <kind>.daily_quota), then env.toml [<kind>]
// daily_quota, then the default here.
const (
	QuotaAIInsight = "ai_insight"
	QuotaGeocoding = "geocoding"

	DefaultAIInsightDailyQuota = 50
	DefaultGeocodingDailyQuota = 100
)

// QuotaSettingKey is the SystemSettings key of a quota kind's daily limit.
func QuotaSettingKey(kind string) string {
	return kind + ".daily_quota"
}

// SiteAIMetrics.MetricKey values (the fixed metric names of the Gemini
// answer), in the order they are stored.
const (
	MetricKeyFootTraffic   = "footTraffic"
	MetricKeyDemographics  = "demographics"
	MetricKeyCompetition   = "competition"
	MetricKeyAccessibility = "accessibility"
	MetricKeyEconomic      = "economic"
)

// MetricKeyOrder is the order the 5 metrics are written in.
var MetricKeyOrder = []string{MetricKeyFootTraffic, MetricKeyDemographics, MetricKeyCompetition, MetricKeyAccessibility, MetricKeyEconomic}

// TaskActionDuplicateConfirmed: the staff sent although other active
// requests are close by (local §8); the note names them.
const TaskActionDuplicateConfirmed = "DUPLICATE_CONFIRMED"

// NearbyRequestExtraStatuses are the "active" statuses besides WAITING_*
// for "other requests nearby" (local §8): sent and not finally rejected.
// Not DRAFT (may be abandoned), not REJECTED. (Local §8 also counted
// REJECTED_BY_DIV_HEAD, which only exists with §5, not ported.)
// HOLD_* counts as active too (November scope, API-CHANGES §13.3).
var NearbyRequestExtraStatuses = []string{TaskStatusRevision, TaskStatusCompleted, TaskStatusHold(RoleDeptHead), TaskStatusHold(RoleDivHead)}

// Duplicate check on submit (config nearby_request.duplicate_check).
const (
	DuplicateCheckOff     = "off"     // default: submit exactly as before
	DuplicateCheckConfirm = "confirm" // 409 unless confirm_duplicate = true
)

// Nearby place categories (local §6), in the order of answers and prompt.
const (
	NearbyCategoryCompetitor = "competitor"
	NearbyCategoryOffice     = "office"
	NearbyCategorySchool     = "school"
	NearbyCategoryUniversity = "university"
	NearbyCategoryApartment  = "apartment"
)

// NearbyCategories is the fixed category order.
var NearbyCategories = []string{NearbyCategoryCompetitor, NearbyCategoryOffice, NearbyCategorySchool, NearbyCategoryUniversity, NearbyCategoryApartment}

// Nearby place data sources.
const (
	NearbySourceGoogle = "google"
	NearbySourceOSM    = "osm"
	NearbySourceMixed  = "mixed" // some categories from Google, the rest from OSM
)
