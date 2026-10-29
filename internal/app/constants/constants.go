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
	// TaskStatusCancelled (Wes, 26 Oct): the staff cancelled a request
	// returned to them (REVISION); closed, kept with its history
	TaskStatusCancelled = "CANCELLED"
)

// TaskStatusRejectedGroup is the list filter value for "Ditolak" (Wes, 26
// Oct): REVISION (back to the staff, "Tolak, bisa direvisi") + REJECTED
// ("Tolak final", closed for good). GET /v1/task?status=REJECTED_ANY.
const TaskStatusRejectedGroup = "REJECTED_ANY"

// RejectedStatuses are the statuses of the "Ditolak" group.
var RejectedStatuses = []string{TaskStatusRevision, TaskStatusRejected}

// InactiveStatuses are the closed requests (Wes, 26 Oct): final rejected
// and cancelled. They are kept (history, Riwayat) but count as nothing
// active: not waiting, not in the duplicate check / other requests nearby.
// The staff app's default list leaves them out with
// status_not=REJECTED,CANCELLED.
var InactiveStatuses = []string{TaskStatusRejected, TaskStatusCancelled}

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

// TaskStatusWaitingGroup is the list filter value "Menunggu" (Wes, 28 Oct,
// simple app): every request waiting for a decision at either level, in one
// paged list. GET /v1/task?status=WAITING.
const TaskStatusWaitingGroup = "WAITING"

// WaitingStatuses are the statuses of the "Menunggu" group. SUBMITED is in
// for safety: a sent request normally moves to WAITING_DEPT_HEAD at once.
var WaitingStatuses = []string{TaskStatusSubmited, TaskStatusWaiting(RoleDeptHead), TaskStatusWaiting(RoleDivHead)}

// CompletedStatuses are what status=COMPLETED ("Disetujui") answers (Wes,
// 28 Oct, simple app): COMPLETED plus APPROVED, which this backend never
// writes on a task but old site-dev-api data might have.
var CompletedStatuses = []string{TaskStatusCompleted, TaskStatusApproved}

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
	// November scope (API-CHANGES §13.3): staff send a REVISION task again
	// (was a second SUBMIT)
	TaskActionResubmit = "RESUBMIT"
	// TaskActionCancel (Wes, 26 Oct): the staff cancelled a REVISION
	// request ("Batalkan pengajuan")
	TaskActionCancel = "CANCEL"
)

// UserFeedSeen.Feed values (November scope): the staff's Activity Log and
// the approver's "Baru" news.
const (
	FeedActivity     = "ACTIVITY"
	FeedApproverNews = "APPROVER_NEWS"
)

// ActivityFeedActions are the history actions shown in the Activity Log
// (API-CHANGES §13.4; not CREATE/UPDATE/FORWARD/COMPLETE).
// + CANCEL (Wes, 26 Oct: the heads see "Dibatalkan oleh staf").
var ActivityFeedActions = []string{TaskActionSubmit, TaskActionResubmit, TaskActionApprove, TaskActionRevision, TaskActionHold, TaskActionRelease, TaskActionReject, TaskActionCancel}

// Activity filters (GET /v1/activity?filter=).
const (
	ActivityFilterAll    = "all"
	ActivityFilterAction = "action" // needs my action: newest row per such request
	ActivityFilterMine   = "mine"   // the approvers' decisions on my requests
)

// Approver news kinds (GET /v1/task/approver-news).
const (
	ApproverNewsNewTask         = "NEW_TASK"
	ApproverNewsDivHeadReturned = "DIV_HEAD_RETURNED"
)

// FeedNeverSeenWindowHours: a feed never opened counts the last 3 days as
// new (DEFAULT, as the prototype's first open; open for Wes).
const FeedNeverSeenWindowHours = 72

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

// SiteContexts.VehicleTrafficLevel values (November scope, API-CHANGES
// §13.5): Sepi, Sedang, Ramai, Sangat ramai.
const (
	VehicleTrafficQuiet    = "QUIET"
	VehicleTrafficModerate = "MODERATE"
	VehicleTrafficBusy     = "BUSY"
	VehicleTrafficVeryBusy = "VERY_BUSY"
)

// NotesMax: approver / staff notes (catatan) on a decision or a cancel, in
// characters (site-dev-api max=500).
const NotesMax = 500

// Survey field limits (November scope, API-CHANGES §13.5 / §6 note), in
// characters.
const (
	VehicleTrafficCountMax   = 99999
	VehicleTrafficMinutesMin = 1
	VehicleTrafficMinutesMax = 120
	VehicleTrafficNoteMax    = 300
	ParkingNoteMax           = 300
	CompetitorNoteMax        = 500
)

// ID prefixes: Tasks.TaskID = "TS" + YYYYMMDD + 6 digits. SiteIDPrefix is
// the OLD site ID ("ST" + YYYYMMDD + 6 digits, sites made before 27 Oct);
// new sites get "SDdd/mm/yyyy" + 3 digits (site_id.go).
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
	// Wes, 26 Oct: cancelled by the staff (still "all except APPROVED")
	TaskStatusCancelled,
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

// AI analysis limits on top of the per-user daily quota (Wes, 23 Oct),
// env.toml [ai_insight]; 0 or missing = off (the default, as before):
//   - site_quota: Gemini answers per site location (~110 m cell, all
//     users, every round incl. resubmits; kept 365 days);
//   - daily_cap: Gemini answers per day (Asia/Jakarta), all users.
//
// Only answered calls count (a failed Gemini call gives the count back).
const (
	AILimitSite  = "site"
	AILimitTotal = "total"

	AIInsightSiteQuotaKey = "ai_insight.site_quota"
	AIInsightDailyCapKey  = "ai_insight.daily_cap"

	// AIInsightAutoGenerateOnSaveKey (Wes, 26 Oct): true or missing = a
	// save without insight/generate_ai starts an AI analysis by the old
	// rules; false = only when the app sends generate_ai: true.
	AIInsightAutoGenerateOnSaveKey = "ai_insight.auto_generate_on_save"
)

// SiteAIMetrics.MetricKey values (the fixed metric names of the Gemini
// answer), in the order they are stored.
const (
	MetricKeyFootTraffic   = "footTraffic"
	MetricKeyDemographics  = "demographics"
	MetricKeyCompetition   = "competition"
	MetricKeyAccessibility = "accessibility"
	MetricKeyEconomic      = "economic"
	// MetricKeyVehicleTraffic "Lalu Lintas Kendaraan": the 6th metric
	// (November scope, API-CHANGES §13.5), one more SiteAIMetrics row
	MetricKeyVehicleTraffic = "vehicleTraffic"
)

// MetricKeyOrder is the order the metrics are written in (the 6th only
// when the insight has it).
var MetricKeyOrder = []string{MetricKeyFootTraffic, MetricKeyDemographics, MetricKeyCompetition, MetricKeyAccessibility, MetricKeyEconomic, MetricKeyVehicleTraffic}

// Overall score weights with the 6th metric (November scope, API-CHANGES
// §13.5): round(0.18 × the 5 old metrics + 0.10 × vehicleTraffic). DEFAULT
// chosen by the agent, open for Wes (review §5 #14). They are written into
// the Gemini prompt.
const (
	ScoreWeightMainMetric     = "0.18"
	ScoreWeightVehicleTraffic = "0.10"
)

// Overall score from ALL inputs (Wes, 27 Oct), computed by the server
// (client.ScoreInsight), not taken from Gemini:
//
//	overall = clamp(round(0.18 × 5 metrics + 0.10 × vehicleTraffic)
//	                − risk penalty + opportunity bonus, 0, 100)
//
// The 6 metrics already carry the survey (parking, notes, competitor note,
// map data: prompt rule I). Risk penalty per risk Gemini lists: high 4,
// moderate 2, low 1, at most 10. Opportunity bonus: Gemini's
// opportunityBonus (+1 per concrete, input-backed opportunity), 0..5.
const (
	ScoreRiskPenaltyHigh     = 4
	ScoreRiskPenaltyModerate = 2
	ScoreRiskPenaltyLow      = 1
	ScoreRiskPenaltyMax      = 10
	ScoreOpportunityBonusMax = 5
)

// TaskActionDuplicateConfirmed: the staff sent although other active
// requests are close by (local §8); the note names them.
const TaskActionDuplicateConfirmed = "DUPLICATE_CONFIRMED"

// NearbyRequestExtraStatuses are the "active" statuses besides WAITING_*
// for "other requests nearby" (local §8): sent and not finally rejected.
// Not DRAFT (may be abandoned), not REJECTED. (Local §8 also counted
// REJECTED_BY_DIV_HEAD, which only exists with §5, not ported.)
// HOLD_* counts as active too (November scope, API-CHANGES §13.3).
// CANCELLED (Wes, 26 Oct) is not active either, like REJECTED.
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

// NearbyCategoryOtherStore (Wes, 29 Oct): convenience stores / minimarkets
// that are NOT one of the active CompetitorBrands, so the Site division can
// spot new competitors. Not a places.categories value: it is taken from the
// competitor search already made (no extra Google call) and answered only
// when asked (GET /v1/site/nearby-places?include=other_store) and switched
// on ([places] other_stores = true). Not in the AI prompt.
const NearbyCategoryOtherStore = "other_store"

// NearbyCategories is the fixed category order.
var NearbyCategories = []string{NearbyCategoryCompetitor, NearbyCategoryOffice, NearbyCategorySchool, NearbyCategoryUniversity, NearbyCategoryApartment}

// Nearby place data sources.
const (
	NearbySourceGoogle = "google"
	NearbySourceOSM    = "osm"
	NearbySourceMixed  = "mixed" // some categories from Google, the rest from OSM
)
