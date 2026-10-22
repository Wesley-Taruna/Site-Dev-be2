package dto

import (
	"fmt"
	"time"
	"unicode/utf8"

	"gitlab.com/fami-app-v2/familib/familabs/validation"
)

// Portal status groups (November scope, API-CHANGES §11 + 20/21 Oct, exact
// values): APPROVED = COMPLETED + APPROVED, WAITING_DEPT_HEAD,
// WAITING_DIV_HEAD, HOLD = both HOLD_*, REJECTED = REJECTED + REVISION.
// Drafts never appear.
const (
	PortalGroupWaitingDeptHead = "WAITING_DEPT_HEAD"
	PortalGroupWaitingDivHead  = "WAITING_DIV_HEAD"
	PortalGroupHold            = "HOLD"
	PortalGroupApproved        = "APPROVED"
	PortalGroupRejected        = "REJECTED"
)

// PortalSiteRequestGet is the query of GET /v1/portal/site.
type PortalSiteRequestGet struct {
	Status string // one group, "" = all
	Search string // site name / ID, any length
	Offset int
	Limit  int
}

// SiteAdsResponse is the ADS data of a site (prototype SiteAds).
type SiteAdsResponse struct {
	AdsValue      *int64     `json:"ads_value"`
	AdsNote       string     `json:"ads_note"`
	UpdatedByName string     `json:"updated_by_name"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

// PortalSiteResponseGet is one row of the portal site log (prototype
// PortalSiteRow).
type PortalSiteResponseGet struct {
	SiteID         string           `json:"site_id"`
	TaskID         string           `json:"task_id"`
	SiteName1      string           `json:"site_name_1"`
	SiteAddress1   string           `json:"site_address_1"`
	SiteAddress2   string           `json:"site_address_2"`
	SiteDistrict   string           `json:"site_district"`
	SiteCity       string           `json:"site_city"`
	SiteRegion     string           `json:"site_region"`
	SitePostalCode string           `json:"site_postal_code"`
	SiteLat        float64          `json:"site_lat"`
	SiteLon        float64          `json:"site_lon"`
	TaskStatus     string           `json:"task_status"`
	StatusGroup    string           `json:"status_group"`
	HoldNote       *string          `json:"hold_note,omitempty"`
	CreatedBy      int64            `json:"created_by"`
	CreatedByName  string           `json:"created_by_name"`
	CreatedByNIK   string           `json:"created_by_nik"`
	SubmittedAt    *time.Time       `json:"submitted_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	Ads            *SiteAdsResponse `json:"ads"`
}

// PortalStatusCountResponseGet is one group of GET
// /v1/portal/site/status/count.
type PortalStatusCountResponseGet struct {
	StatusGroup string `json:"status_group"`
	Count       int64  `json:"count"`
}

// PortalReportSite is the site of the report with its creator.
type PortalReportSite struct {
	SiteResponseGet
	CreatedByNIK string `json:"created_by_nik"`
}

// PortalReportResponseGet is GET /v1/portal/site/:id/report (prototype
// PortalReportResp): everything the PDF report needs in one call.
type PortalReportResponseGet struct {
	Site        PortalReportSite            `json:"site"`
	Context     *SiteContextResponseGet     `json:"context"`
	Task        TaskDetailTaskSection       `json:"task"`
	Attachment  []TaskAttachmentResponseGet `json:"attachment"`
	Insight     *AIInsightResponseGet       `json:"insight"`
	StaffNote   string                      `json:"staff_note"`
	Ads         *SiteAdsResponse            `json:"ads"`
	StatusGroup *string                     `json:"status_group"`
}

// SiteAdsMaxValue is the highest ADS value accepted (as the prototype).
const SiteAdsMaxValue = 10_000_000_000

// SiteAdsMaxNote is the longest ADS note (characters).
const SiteAdsMaxNote = 300

// SiteAdsRequestUpdate is the body of PUT /v1/portal/site/:id/ads:
// ads_value whole Rupiah >= 0 or null, note at most 300 characters.
type SiteAdsRequestUpdate struct {
	AdsValue *int64 `json:"ads_value"`
	AdsNote  string `json:"ads_note"`
}

// Validate implements validation.Request.
func (r SiteAdsRequestUpdate) Validate() *validation.Output {
	o := &validation.Output{Valid: true}
	if v := r.AdsValue; v != nil && (*v < 0 || *v > SiteAdsMaxValue) {
		o.Failure("ads_value.between", fmt.Sprintf("The ads value must be between 0 and %d.", int64(SiteAdsMaxValue)))
	}
	if utf8.RuneCountInString(r.AdsNote) > SiteAdsMaxNote {
		o.Failure("ads_note.lte", fmt.Sprintf("The ads note may not be greater than %d characters.", SiteAdsMaxNote))
	}
	return o
}

// Messages implements validation.Request.
func (r SiteAdsRequestUpdate) Messages() map[string]string {
	return map[string]string{}
}
