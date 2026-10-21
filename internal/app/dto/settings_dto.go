package dto

import (
	"time"

	"gitlab.com/fami-app-v2/familib/familabs/validation"
)

// LimitsResponseGet is GET/PUT /v1/settings/limits (site-dev-api LimitsResp).
type LimitsResponseGet struct {
	AIInsightDailyQuota    int64      `json:"ai_insight_daily_quota"`
	AIInsightUpdatedBy     int64      `json:"ai_insight_updated_by,omitempty"`
	AIInsightUpdatedByName string     `json:"ai_insight_updated_by_name,omitempty"`
	AIInsightUpdatedAt     *time.Time `json:"ai_insight_updated_at,omitempty"`
	GeocodingDailyQuota    int64      `json:"geocoding_daily_quota"`
	GeocodingUpdatedBy     int64      `json:"geocoding_updated_by,omitempty"`
	GeocodingUpdatedByName string     `json:"geocoding_updated_by_name,omitempty"`
	GeocodingUpdatedAt     *time.Time `json:"geocoding_updated_at,omitempty"`
}

// LimitsRequestUpdate is the body of PUT /v1/settings/limits; a missing
// value is left unchanged, a given value must be at least 1.
type LimitsRequestUpdate struct {
	AIInsightDailyQuota *int64 `json:"ai_insight_daily_quota"`
	GeocodingDailyQuota *int64 `json:"geocoding_daily_quota"`
}

// Validate implements validation.Request (rule "at least 1 when given").
func (r LimitsRequestUpdate) Validate() *validation.Output {
	o := &validation.Output{Valid: true}
	if r.AIInsightDailyQuota != nil && *r.AIInsightDailyQuota < 1 {
		o.Failure("ai_insight_daily_quota.gte", "The ai insight daily quota should be greater than 1")
	}
	if r.GeocodingDailyQuota != nil && *r.GeocodingDailyQuota < 1 {
		o.Failure("geocoding_daily_quota.gte", "The geocoding daily quota should be greater than 1")
	}
	return o
}

// Messages implements validation.Request.
func (r LimitsRequestUpdate) Messages() map[string]string {
	return map[string]string{}
}

// QuotaFlushRequest is the body of POST /v1/settings/quota-flush.
type QuotaFlushRequest struct {
	UserID    int64  `json:"user_id" valid:"required"`
	QuotaType string `json:"quota_type" valid:"required|in:ai_insight,geocoding"`
}

// QuotaFlushResponse is the answer of POST /v1/settings/quota-flush
// (site-dev-api QuotaFlushResp).
type QuotaFlushResponse struct {
	UserID    int64  `json:"user_id"`
	QuotaType string `json:"quota_type"`
	Remaining int64  `json:"remaining"`
	Limit     int64  `json:"limit"`
}
