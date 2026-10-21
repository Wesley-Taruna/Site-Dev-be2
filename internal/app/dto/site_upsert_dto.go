package dto

import (
	"time"

	"gitlab.com/fami-app-v2/familib/familabs/validation"
)

// SiteRequestForm is the site part of POST /v1/task/:status: site_id empty
// = new site, filled = update (site-dev-api SiteCreateReq).
type SiteRequestForm struct {
	SiteID                 string   `json:"site_id"`
	SiteName1              string   `json:"site_name_1" valid:"required"`
	SiteName2              string   `json:"site_name_2"`
	SiteAddress1           string   `json:"site_address_1"`
	SiteAddress2           string   `json:"site_address_2"`
	SiteAddress3           string   `json:"site_address_3"`
	SitePostalCode         string   `json:"site_postal_code"`
	SiteDistrict           string   `json:"site_district"`
	SiteCity               string   `json:"site_city"`
	SiteRegion             string   `json:"site_region"`
	SiteLat                float64  `json:"site_lat"`
	SiteLon                float64  `json:"site_lon"`
	SiteTelepon            string   `json:"site_telepon"`
	SiteTeleponExt         string   `json:"site_telepon_ext"`
	SiteMobilePhone        string   `json:"site_mobile_phone"`
	SitePic                string   `json:"site_pic"`
	SiteFax                string   `json:"site_fax"`
	SitePlannedOpeningDate DateOnly `json:"site_planned_opening_date"`
	SiteSalesArea          float64  `json:"site_sales_area"`
	SiteSellingAlcohol     bool     `json:"site_selling_alcohol"`
	SiteOpen24Hours        bool     `json:"site_open_24_hours"`
}

// AttachmentItem is one file already uploaded (POST /v1/upload/add).
type AttachmentItem struct {
	FileName string `json:"file_name" valid:"required"`
	Path     string `json:"path" valid:"required"`
}

// SiteRequestUpsert is the body of POST /v1/task/:status (status DRAFT or
// SUBMITED): site, optional survey, attachments (replace all when sent),
// history note, and an insight made before by preview (stored as is,
// no new Gemini call). Site-dev-api SiteCreateWithContextReq.
type SiteRequestUpsert struct {
	Site       SiteRequestForm           `json:"site"`
	Context    *SiteContextRequestUpsert `json:"context"`
	Attachment []AttachmentItem          `json:"attachment"`
	Notes      string                    `json:"notes"`
	Insight    *AIInsightResponseGet     `json:"insight"`
}

// Validate implements validation.Request (notes at most 500 characters).
func (r SiteRequestUpsert) Validate() *validation.Output {
	o := &validation.Output{Valid: true}
	if len(r.Notes) > 500 {
		o.Failure("notes.lte", "The notes may not be greater than 500")
	}
	return o
}

// Messages implements validation.Request.
func (r SiteRequestUpsert) Messages() map[string]string {
	return map[string]string{}
}

// AttachmentResponse is one saved attachment (site-dev-api AttachmentResp).
type AttachmentResponse struct {
	SiteAttachmentID uint      `json:"site_attachment_id"`
	FileName         string    `json:"file_name"`
	Path             string    `json:"path"`
	CreatedAt        time.Time `json:"created_at"`
}

// SiteUpsertResponse is the answer of POST /v1/task/:status (site-dev-api
// SiteUpsertResp).
type SiteUpsertResponse struct {
	Site        SiteResponseGet         `json:"site"`
	Context     *SiteContextResponseGet `json:"context,omitempty"`
	Attachments []AttachmentResponse    `json:"attachments,omitempty"`
	Task        TaskDetailResponseGet   `json:"task"`
}
