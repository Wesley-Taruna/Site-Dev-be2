package router

// Overall score from all inputs + "why this score" (Wes, 27 Oct): stored
// with the insight (ScoreReason, ScoreBreakdown) and read back; additive
// fields, older insights answer exactly as before.

import (
	"database/sql/driver"
	"net/http"
	"strings"
	"testing"

	"site-dev-backend/internal/app/client"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func scoredGeminiInsight() dto.AIInsightResponseGet {
	ins := geminiInsight()
	vt := metric("Lalu", 80)
	ins.Metrics.VehicleTraffic = &vt
	ins.ScoreReason = "Plus terbesar: akses (85) dan keramaian (80). Minus terbesar: parkir terbatas."
	client.ScoreInsight(&ins, 1)
	return ins
}

func TestAIScore_GenerateStoresReasonAndBreakdown(t *testing.T) {
	gemini := withGemini(t)
	withRedis(t)
	gemini.On("GenerateInsight", mock.Anything, mock.Anything).Return(scoredGeminiInsight(), "gemini-2.5-flash", nil)
	h, rec, _ := stack(t, append([]sqlfake.Result{{Match: `INSERT INTO "SiteAIInsights"`, Columns: []string{"SiteAIID"}, Rows: [][]driver.Value{{int64(78)}}}},
		detailResults("USER")...)...)

	code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight", map[string]any{"site_id": "ST1"}, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	// base round(0.18 × 370 + 0.10 × 80) = 75, moderate risk −2, bonus +1 = 74
	var got dto.AIInsightResponseGet
	require.NoError(t, jsonUnmarshalData(body, &got))
	assert.Equal(t, 74, got.OverallScore)
	assert.Equal(t, &dto.AIScoreBreakdown{Base: 75, RiskPenalty: 2, OpportunityBonus: 1}, got.ScoreBreakdown)
	assert.Contains(t, string(body), `"score_reason":"Plus terbesar`)
	var insert string
	for _, s := range rec.sqls {
		if strings.HasPrefix(s, `INSERT INTO "SiteAIInsights"`) {
			insert = s
		}
	}
	assert.Contains(t, insert, `'Plus terbesar: akses (85) dan keramaian (80). Minus terbesar: parkir terbatas.','{"base":75,"risk_penalty":2,"opportunity_bonus":1}'`)
}

func TestAIScore_GetReturnsReasonAndBreakdown(t *testing.T) {
	h, _, _ := stack(t, append([]sqlfake.Result{
		{Match: `FROM "SiteAIInsights"`, Columns: []string{"SiteAIID", "SiteID", "OverallScore", "Recommendation", "Headline", "Narrative", "ConfidenceLevel", "ConfidenceReason", "GeminiModel", "GeneratedBy", "GeneratedAt", "ScoreReason", "ScoreBreakdown"},
			Rows: [][]driver.Value{{int64(78), "ST1", int64(74), "Recommended", "Toko Kemang — Recommended (74/100)", "Layak.", "medium", "x", "gemini-2.5-flash", int64(1001), generatedAt,
				"Plus: akses. Minus: parkir.", `{"base":75,"risk_penalty":2,"opportunity_bonus":1}`}}},
	}, detailResults("USER")...)...)

	code, body := call(t, h, http.MethodGet, "/v1/task/ai-insight/ST1", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Contains(t, string(body), `"generatedAt":"2026-10-21T09:30:00Z","score_reason":"Plus: akses. Minus: parkir.","score_breakdown":{"base":75,"risk_penalty":2,"opportunity_bonus":1}`)
}
