package dto

import (
	"time"

	"gitlab.com/fami-app-v2/familib/familabs/validation"
)

// AIInsightRequestGenerate is the body of POST /v1/task/ai-insight.
type AIInsightRequestGenerate struct {
	SiteID string `json:"site_id" valid:"required"`
}

// AIInsightQuotaResponse is GET /v1/task/ai-insight/quota (site-dev-api
// AIInsightQuotaResp).
type AIInsightQuotaResponse struct {
	Limit     int64 `json:"limit"`
	Remaining int64 `json:"remaining"`
}

// PreviewSiteRequest is the site part of the preview: the site form without
// site_id; only the coordinates are required.
type PreviewSiteRequest struct {
	SiteName1              string   `json:"site_name_1"`
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
	SitePlannedOpeningDate DateOnly `json:"site_planned_opening_date"`
	SiteSalesArea          float64  `json:"site_sales_area"`
	SiteSellingAlcohol     bool     `json:"site_selling_alcohol"`
	SiteOpen24Hours        bool     `json:"site_open_24_hours"`
}

// Validate implements validation.Request: both coordinates are required
// (non-zero, like site-dev-api's go-playground "required"; familabs'
// "required" accepts 0 for floats).
func (r PreviewSiteRequest) Validate() *validation.Output {
	o := &validation.Output{Valid: true}
	if r.SiteLat == 0 {
		o.Failure("site_lat.required", "The site lat field is required.")
	}
	if r.SiteLon == 0 {
		o.Failure("site_lon.required", "The site lon field is required.")
	}
	return o
}

// Messages implements validation.Request.
func (r PreviewSiteRequest) Messages() map[string]string {
	return map[string]string{}
}

// AIInsightRequestPreview is the body of POST /v1/task/ai-insight/preview:
// site + survey not saved yet; nothing is stored.
type AIInsightRequestPreview struct {
	Site    PreviewSiteRequest        `json:"site"`
	Context *SiteContextRequestUpsert `json:"context"`
}

// AIInsightMetric / AIInsightMetrics / AIInsightCompetitor / AIInsightRisk /
// AIInsightContextWarning / AIInsightResponseGet are the Gemini insight
// (site-dev-api dto/site_ai_insight.go, JSON names in camelCase as there).
type AIInsightMetric struct {
	Label   string   `json:"label"`
	Score   int      `json:"score"`
	Band    string   `json:"band"`
	Summary string   `json:"summary"`
	Factors []string `json:"factors"`
}

type AIInsightMetrics struct {
	FootTraffic   AIInsightMetric `json:"footTraffic"`
	Demographics  AIInsightMetric `json:"demographics"`
	Competition   AIInsightMetric `json:"competition"`
	Accessibility AIInsightMetric `json:"accessibility"`
	Economic      AIInsightMetric `json:"economic"`
	// VehicleTraffic "Lalu Lintas Kendaraan" (November scope, API-CHANGES
	// §13.5); left out on insights made before
	VehicleTraffic *AIInsightMetric `json:"vehicleTraffic,omitempty"`
}

type AIInsightCompetitor struct {
	Name       string  `json:"name"`
	DistanceKm float64 `json:"distanceKm"`
	Type       string  `json:"type"`
}

type AIInsightRisk struct {
	Title      string `json:"title"`
	Severity   string `json:"severity"`
	Mitigation string `json:"mitigation"`
}

type AIInsightContextWarning struct {
	Field         string `json:"field"`
	UserClaim     string `json:"userClaim"`
	AIExpectation string `json:"aiExpectation"`
	Severity      string `json:"severity"`
	Message       string `json:"message"`
}

type AIInsightResponseGet struct {
	SiteAIID         int64                     `json:"site_ai_id"`
	SiteID           string                    `json:"site_id"`
	OverallScore     int                       `json:"overallScore"`
	Recommendation   string                    `json:"recommendation"`
	Headline         string                    `json:"headline"`
	Narrative        string                    `json:"narrative"`
	Metrics          AIInsightMetrics          `json:"metrics"`
	Competitors      []AIInsightCompetitor     `json:"competitors"`
	Risks            []AIInsightRisk           `json:"risks"`
	Opportunities    []string                  `json:"opportunities"`
	ConfidenceLevel  string                    `json:"confidenceLevel"`
	ConfidenceReason string                    `json:"confidenceReason"`
	DataAssumptions  []string                  `json:"dataAssumptions"`
	ContextWarnings  []AIInsightContextWarning `json:"contextWarnings"`
	GeneratedAt      time.Time                 `json:"generatedAt"`
	// DataSource / Measured (optional, local §6): which map data the AI got
	// ("gemini+google", "gemini+osm", "gemini+mixed", or "gemini" when the
	// lookup failed). Only on a fresh generate/preview while [places] is
	// enabled; not stored.
	DataSource string          `json:"data_source,omitempty"`
	Measured   *NearbyMeasured `json:"measured,omitempty"`
}

// AIInsightTaskPayload is the queue job of POST /v1/queue/ai-insight.
type AIInsightTaskPayload struct {
	SiteID  string `json:"site_id"`
	ActorID int64  `json:"actor_id"`
}
