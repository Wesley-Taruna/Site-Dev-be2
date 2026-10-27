package model

import (
	"encoding/json"
	"time"
)

// SiteAIInsight is one Gemini insight of a site. Since 27 Oct (Wes) a new
// one is added and the older ones stay as history; the newest (highest
// SiteAIID) is the one shown and sent. Metrics, risks, opportunities, data
// assumptions and context warnings are child tables; competitors are not
// stored (they come from SiteCompetitors).
type SiteAIInsight struct {
	SiteAIID         int64     `gorm:"column:SiteAIID;primaryKey" json:"site_ai_id"`
	SiteID           string    `gorm:"column:SiteID" json:"site_id"`
	OverallScore     int       `gorm:"column:OverallScore" json:"overall_score"`
	Recommendation   string    `gorm:"column:Recommendation" json:"recommendation"`
	Headline         string    `gorm:"column:Headline" json:"headline"`
	Narrative        string    `gorm:"column:Narrative" json:"narrative"`
	ConfidenceLevel  string    `gorm:"column:ConfidenceLevel" json:"confidence_level"`
	ConfidenceReason string    `gorm:"column:ConfidenceReason" json:"confidence_reason"`
	GeminiModel      string    `gorm:"column:GeminiModel" json:"gemini_model"`
	GeneratedBy      int64     `gorm:"column:GeneratedBy" json:"generated_by"`
	GeneratedAt      time.Time `gorm:"column:GeneratedAt" json:"generated_at"`
	CreatedAt        time.Time `gorm:"column:CreatedAt" json:"created_at"`
	// Wes, 27 Oct (new columns, plan §8): "why this score" and how the
	// server computed it (JSON of dto.AIScoreBreakdown); NULL on older rows
	ScoreReason    *string `gorm:"column:ScoreReason" json:"score_reason"`
	ScoreBreakdown *string `gorm:"column:ScoreBreakdown" json:"score_breakdown"`
	// InputHash (Wes, 27 Oct, new column): fingerprint of the AI inputs it
	// was made from (aiInputHash); a re-run with the same inputs is refused
	InputHash *string `gorm:"column:InputHash" json:"input_hash"`
}

func (SiteAIInsight) TableName() string {
	return "SiteAIInsights"
}

func (m *SiteAIInsight) MarshalJSON() ([]byte, error) {
	type Alias SiteAIInsight

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}

// SiteAIMetric is one of the 5 metrics of an insight (MetricKey =
// constants.MetricKey...). Factors are joined with ";".
type SiteAIMetric struct {
	ID        int64     `gorm:"column:ID;primaryKey" json:"id"`
	SiteAIID  int64     `gorm:"column:SiteAIID" json:"site_ai_id"`
	MetricKey string    `gorm:"column:MetricKey" json:"metric_key"`
	Label     string    `gorm:"column:Label" json:"label"`
	Score     int       `gorm:"column:Score" json:"score"`
	Band      string    `gorm:"column:Band" json:"band"`
	Summary   string    `gorm:"column:Summary" json:"summary"`
	Factors   string    `gorm:"column:Factors" json:"factors"`
	CreatedAt time.Time `gorm:"column:CreatedAt" json:"created_at"`
}

func (SiteAIMetric) TableName() string {
	return "SiteAIMetrics"
}

// SiteAIRisk is one risk of an insight.
type SiteAIRisk struct {
	ID         int64     `gorm:"column:ID;primaryKey" json:"id"`
	SiteAIID   int64     `gorm:"column:SiteAIID" json:"site_ai_id"`
	Title      string    `gorm:"column:Title" json:"title"`
	Severity   string    `gorm:"column:Severity" json:"severity"`
	Mitigation string    `gorm:"column:Mitigation" json:"mitigation"`
	CreatedAt  time.Time `gorm:"column:CreatedAt" json:"created_at"`
}

func (SiteAIRisk) TableName() string {
	return "SiteAIRisks"
}

// SiteAIOpportunity is one opportunity of an insight.
type SiteAIOpportunity struct {
	ID          int64     `gorm:"column:ID;primaryKey" json:"id"`
	SiteAIID    int64     `gorm:"column:SiteAIID" json:"site_ai_id"`
	Opportunity string    `gorm:"column:Opportunity" json:"opportunity"`
	CreatedAt   time.Time `gorm:"column:CreatedAt" json:"created_at"`
}

func (SiteAIOpportunity) TableName() string {
	return "SiteAIOpportunities"
}

// SiteAIDataAssumption is one data assumption of an insight.
type SiteAIDataAssumption struct {
	ID         int64     `gorm:"column:ID;primaryKey" json:"id"`
	SiteAIID   int64     `gorm:"column:SiteAIID" json:"site_ai_id"`
	Assumption string    `gorm:"column:Assumption" json:"assumption"`
	CreatedAt  time.Time `gorm:"column:CreatedAt" json:"created_at"`
}

func (SiteAIDataAssumption) TableName() string {
	return "SiteAIDataAssumptions"
}

// SiteAIContextWarning is one doubt Gemini raised about a survey claim.
type SiteAIContextWarning struct {
	ID            int64     `gorm:"column:ID;primaryKey" json:"id"`
	SiteAIID      int64     `gorm:"column:SiteAIID" json:"site_ai_id"`
	Field         string    `gorm:"column:Field" json:"field"`
	UserClaim     string    `gorm:"column:UserClaim" json:"user_claim"`
	AIExpectation string    `gorm:"column:AIExpectation" json:"ai_expectation"`
	Severity      string    `gorm:"column:Severity" json:"severity"`
	Message       string    `gorm:"column:Message" json:"message"`
	CreatedAt     time.Time `gorm:"column:CreatedAt" json:"created_at"`
}

func (SiteAIContextWarning) TableName() string {
	return "SiteAIContextWarnings"
}
