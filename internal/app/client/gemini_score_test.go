package client

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scoredMetrics(ft, dem, comp, acc, eco, vt int) dto.AIInsightMetrics {
	m := func(s int) dto.AIInsightMetric { return dto.AIInsightMetric{Score: s} }
	v := m(vt)
	return dto.AIInsightMetrics{FootTraffic: m(ft), Demographics: m(dem), Competition: m(comp), Accessibility: m(acc), Economic: m(eco), VehicleTraffic: &v}
}

func risks(sev ...string) []dto.AIInsightRisk {
	out := make([]dto.AIInsightRisk, 0, len(sev))
	for _, s := range sev {
		out = append(out, dto.AIInsightRisk{Title: s, Severity: s})
	}
	return out
}

// The overall score from all inputs (Wes, 27 Oct): base from the 6 metrics,
// minus the risk penalty (high 4 / moderate 2 / low 1, max 10), plus the
// opportunity bonus (0..5), clamped 0..100; recommendation from the band.
func TestScoreInsight(t *testing.T) {
	for name, tc := range map[string]struct {
		metrics dto.AIInsightMetrics
		risks   []dto.AIInsightRisk
		bonus   int
		want    dto.AIScoreBreakdown
		score   int
		rec     string
	}{
		"metrics only": {scoredMetrics(75, 70, 60, 78, 72, 80), nil, 0, dto.AIScoreBreakdown{Base: 72}, 72, "Recommended"},
		"one moderate risk": {scoredMetrics(75, 70, 60, 78, 72, 80), risks("moderate"), 0,
			dto.AIScoreBreakdown{Base: 72, RiskPenalty: 2}, 70, "Recommended"},
		"risks push below a band": {scoredMetrics(75, 70, 60, 78, 72, 80), risks("high", "moderate"), 0,
			dto.AIScoreBreakdown{Base: 72, RiskPenalty: 6}, 66, "Proceed With Caution"},
		"penalty capped at 10": {scoredMetrics(75, 70, 60, 78, 72, 80), risks("high", "high", "high", "low"), 0,
			dto.AIScoreBreakdown{Base: 72, RiskPenalty: 10}, 62, "Proceed With Caution"},
		"bonus lifts a band": {scoredMetrics(80, 80, 80, 80, 80, 80), risks("low"), 3,
			dto.AIScoreBreakdown{Base: 80, RiskPenalty: 1, OpportunityBonus: 3}, 82, "Strongly Recommended"},
		"bonus clamped to 5": {scoredMetrics(50, 50, 50, 50, 50, 50), nil, 9,
			dto.AIScoreBreakdown{Base: 50, OpportunityBonus: 5}, 55, "Proceed With Caution"},
		"negative bonus = 0": {scoredMetrics(50, 50, 50, 50, 50, 50), nil, -3, dto.AIScoreBreakdown{Base: 50}, 50, "Reconsider"},
		"clamped at 100":     {scoredMetrics(100, 100, 100, 100, 100, 100), nil, 5, dto.AIScoreBreakdown{Base: 100, OpportunityBonus: 5}, 100, "Strongly Recommended"},
		"clamped at 0":       {scoredMetrics(0, 0, 0, 0, 0, 0), risks("high"), 0, dto.AIScoreBreakdown{Base: 0, RiskPenalty: 4}, 0, "Reconsider"},
		"metric out of range clamped": {scoredMetrics(150, 70, 60, 78, 72, -20), nil, 0,
			dto.AIScoreBreakdown{Base: 68}, 68, "Proceed With Caution"}, // 150 → 100, −20 → 0
		"unknown severity ignored": {scoredMetrics(75, 70, 60, 78, 72, 80), risks("critical", "MODERATE"), 0,
			dto.AIScoreBreakdown{Base: 72, RiskPenalty: 2}, 70, "Recommended"},
	} {
		t.Run(name, func(t *testing.T) {
			ins := dto.AIInsightResponseGet{OverallScore: 99, Recommendation: "Strongly Recommended", Metrics: tc.metrics, Risks: tc.risks}

			ScoreInsight(&ins, tc.bonus)

			require.NotNil(t, ins.ScoreBreakdown)
			assert.Equal(t, tc.want, *ins.ScoreBreakdown)
			assert.Equal(t, tc.score, ins.OverallScore)
			assert.Equal(t, tc.rec, ins.Recommendation)
		})
	}
}

func TestScoreInsight_MetricBandsFollowScores(t *testing.T) {
	ins := dto.AIInsightResponseGet{Metrics: scoredMetrics(90, 72, 55, 30, 150, 85)}
	ins.Metrics.FootTraffic.Band = "low" // Gemini's wrong band

	ScoreInsight(&ins, 0)

	assert.Equal(t, "high", ins.Metrics.FootTraffic.Band)
	assert.Equal(t, "good", ins.Metrics.Demographics.Band)
	assert.Equal(t, "moderate", ins.Metrics.Competition.Band)
	assert.Equal(t, "low", ins.Metrics.Accessibility.Band)
	assert.Equal(t, 100, ins.Metrics.Economic.Score)
	assert.Equal(t, "high", ins.Metrics.VehicleTraffic.Band)
}

// The recorded-shape answer: Gemini says 72; server: base 72, moderate
// risk −2, bonus +2 = 72; the reason is kept.
func TestParseGeminiAnswer_ScoreAndReason(t *testing.T) {
	raw, err := os.ReadFile("testdata/gemini_ok.json")
	require.NoError(t, err)

	got, err := ParseGeminiAnswer(raw, &model.Site{SiteName1: "Kemang Timur"}, time.Now())

	require.NoError(t, err)
	assert.Equal(t, 72, got.OverallScore)
	assert.Equal(t, &dto.AIScoreBreakdown{Base: 72, RiskPenalty: 2, OpportunityBonus: 2}, got.ScoreBreakdown)
	assert.Contains(t, got.ScoreReason, "Plus terbesar")
	assert.Equal(t, "Kemang Timur — Recommended (72/100)", got.Headline)

	// the JSON answer: new fields additive, snake_case like data_source
	b, err := json.Marshal(got)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"score_reason":"Plus terbesar`)
	assert.Contains(t, string(b), `"score_breakdown":{"base":72,"risk_penalty":2,"opportunity_bonus":2}`)
}

// A model sum that is off (or a wrong recommendation) never reaches the
// answer.
func TestParseGeminiAnswer_ModelSumIgnored(t *testing.T) {
	inner := `{"overallScore":95,"recommendation":"Strongly Recommended","scoreReason":"x","opportunityBonus":1,` +
		`"metrics":{"footTraffic":{"score":60},"demographics":{"score":60},"competition":{"score":40},"accessibility":{"score":55},"economic":{"score":60},"vehicleTraffic":{"score":50}},` +
		`"risks":[{"title":"Banjir","severity":"high","mitigation":"-"},{"title":"Sewa mahal","severity":"moderate","mitigation":"-"}]}`
	raw, err := json.Marshal(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{map[string]any{"text": inner}}}}}})
	require.NoError(t, err)

	got, err := ParseGeminiAnswer(raw, &model.Site{SiteName1: "Ruko"}, time.Now())

	require.NoError(t, err)
	// base round(0.18 × 275 + 0.10 × 50) = round(54.5) = 55; − 6 + 1 = 50
	assert.Equal(t, 50, got.OverallScore)
	assert.Equal(t, "Reconsider", got.Recommendation)
	assert.Equal(t, "Ruko — Reconsider (50/100)", got.Headline)
}

// A preview answer sent back on save is recomputed (a changed score does
// not stick); an old-shape insight (no breakdown) stays as sent.
func TestRescoreSentInsight(t *testing.T) {
	ins := dto.AIInsightResponseGet{OverallScore: 99, Recommendation: "Strongly Recommended", Headline: "Ruko — Strongly Recommended (99/100)",
		Metrics: scoredMetrics(75, 70, 60, 78, 72, 80), Risks: risks("moderate"), ScoreBreakdown: &dto.AIScoreBreakdown{Base: 90, OpportunityBonus: 7}}

	RescoreSentInsight(&ins)

	assert.Equal(t, 75, ins.OverallScore) // 72 − 2 + 5 (bonus clamped)
	assert.Equal(t, &dto.AIScoreBreakdown{Base: 72, RiskPenalty: 2, OpportunityBonus: 5}, ins.ScoreBreakdown)
	assert.Equal(t, "Ruko — Recommended (75/100)", ins.Headline)

	old := dto.AIInsightResponseGet{OverallScore: 74, Recommendation: "Recommended", Headline: "X — Recommended (74/100)", Metrics: scoredMetrics(1, 1, 1, 1, 1, 1)}
	RescoreSentInsight(&old)
	assert.Equal(t, 74, old.OverallScore)
	assert.Nil(t, old.ScoreBreakdown)
}

func TestPrompt_RuleIAndSchema(t *testing.T) {
	site, ctx, comps, notes := promptFixtureSite()
	p := buildPrompt(site, ctx, comps, notes, nil)
	assert.Contains(t, p, "I) SKOR TOTAL DARI SEMUA INPUT (WAJIB)")
	assert.Contains(t, p, "high −4, moderate −2, low −1; total maksimal −10")
	assert.Contains(t, p, "opportunityBonus (0–5)")
	assert.Contains(t, p, "scoreReason: 2–4 kalimat Bahasa Indonesia")
	schema := string(mustJSON(t, responseSchema))
	assert.Contains(t, schema, `"scoreReason"`)
	assert.Contains(t, schema, `"opportunityBonus"`)
}

// Live runs (27 Oct) showed Gemini writing the total into the reason; the
// server's number wins.
func TestFixReasonScore(t *testing.T) {
	for in, want := range map[string]string{
		"Skor total 64 mencerminkan potensi.":             "Skor total 61 mencerminkan potensi.",
		"Skor keseluruhan 91 mencerminkan lokasi kuat.":   "Skor keseluruhan 61 mencerminkan lokasi kuat.",
		"Skor akhir: 70/100 karena parkir.":               "Skor akhir: 61/100 karena parkir.",
		"Skor keseluruhan yang rendah ini disebabkan ...": "Skor keseluruhan yang rendah ini disebabkan ...",
		"Aksesibilitas (95) dan demografi (90) kuat.":     "Aksesibilitas (95) dan demografi (90) kuat.",
	} {
		assert.Equal(t, want, fixReasonScore(in, 61), in)
	}
	ins := dto.AIInsightResponseGet{Metrics: scoredMetrics(75, 70, 60, 78, 72, 80), Risks: risks("moderate"), ScoreReason: " Skor total 99 karena akses. "}
	ScoreInsight(&ins, 0)
	assert.Equal(t, "Skor total 70 karena akses.", ins.ScoreReason)
}
