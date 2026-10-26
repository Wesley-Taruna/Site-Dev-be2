package dto

import (
	"math"
	"strings"
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
	// ConfirmDuplicate (local §8): the staff saw the 409 DUPLICATE_NEARBY
	// list and still send. Only used with nearby_request.duplicate_check =
	// "confirm".
	ConfirmDuplicate bool `json:"confirm_duplicate"`
	// GenerateAI (Wes, 26 Oct): may the save start an AI analysis when no
	// insight is sent? true = yes (queue job / direct Gemini call, limits
	// apply), false = never, missing/null = [ai_insight]
	// auto_generate_on_save (default true = the old rules: new site, or a
	// draft without an insight yet).
	GenerateAI *bool `json:"generate_ai"`
}

// Fields needed to send (SUBMITED), Wes 26 Oct: only these three; address,
// contacts, opening date, survey (incl. traffic), competitors and photos
// are optional. Keys as in the 400 answer's data.
const (
	SubmitFieldName      = "site_name_1"
	SubmitFieldLocation  = "site_location"
	SubmitFieldSalesArea = "site_sales_area"
)

// submitFieldLabels are the Indonesian names in the 400 message.
var submitFieldLabels = map[string]string{
	SubmitFieldName:      "Nama site",
	SubmitFieldLocation:  "Lokasi (titik di peta)",
	SubmitFieldSalesArea: "Sales Area",
}

// MissingForSubmit lists what a send still needs: a name (not only
// spaces), a valid map point (lat -90..90, lon -180..180, not 0,0) and a
// sales area above 0. Empty = can be sent.
func (f SiteRequestForm) MissingForSubmit() []string {
	missing := []string{}
	if strings.TrimSpace(f.SiteName1) == "" {
		missing = append(missing, SubmitFieldName)
	}
	if !ValidSitePoint(f.SiteLat, f.SiteLon) {
		missing = append(missing, SubmitFieldLocation)
	}
	if !(f.SiteSalesArea > 0) || math.IsInf(f.SiteSalesArea, 0) {
		missing = append(missing, SubmitFieldSalesArea)
	}
	return missing
}

// ValidSitePoint: a real map point (0,0 = never set).
func ValidSitePoint(lat, lon float64) bool {
	if math.IsNaN(lat) || math.IsNaN(lon) || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return false
	}
	return !(lat == 0 && lon == 0)
}

// SubmitMissingMessage is the 400 text, e.g. "Belum bisa dikirim. Lengkapi
// dulu: Lokasi (titik di peta), Sales Area."
func SubmitMissingMessage(missing []string) string {
	labels := make([]string, 0, len(missing))
	for _, m := range missing {
		labels = append(labels, submitFieldLabels[m])
	}
	return "Belum bisa dikirim. Lengkapi dulu: " + strings.Join(labels, ", ") + "."
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
