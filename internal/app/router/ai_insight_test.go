package router

// Side-by-side check, module 8 (AI insight): POST /v1/task/ai-insight,
// /ai-insight/preview, GET /ai-insight/quota, /ai-insight/:site_id against
// site-dev-api 20eeab4. Gemini is a mock here (the real HTTP call is tested
// with recorded-shape fixtures in client/gemini_client_test.go).

import (
	"database/sql/driver"
	"net/http"
	"strings"
	"testing"
	"time"

	"site-dev-backend/internal/app/client"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- copies of site-dev-api 20eeab4 internal/app/dto/site_ai_insight.go ---

type oldAIInsightQuotaResp struct {
	Limit     int64 `json:"limit"`
	Remaining int64 `json:"remaining"`
}

type oldAIInsightMetric struct {
	Label   string   `json:"label"`
	Score   int      `json:"score"`
	Band    string   `json:"band"`
	Summary string   `json:"summary"`
	Factors []string `json:"factors"`
}

type oldAIInsightMetrics struct {
	FootTraffic   oldAIInsightMetric `json:"footTraffic"`
	Demographics  oldAIInsightMetric `json:"demographics"`
	Competition   oldAIInsightMetric `json:"competition"`
	Accessibility oldAIInsightMetric `json:"accessibility"`
	Economic      oldAIInsightMetric `json:"economic"`
}

type oldAIInsightCompetitor struct {
	Name       string  `json:"name"`
	DistanceKm float64 `json:"distanceKm"`
	Type       string  `json:"type"`
}

type oldAIInsightRisk struct {
	Title      string `json:"title"`
	Severity   string `json:"severity"`
	Mitigation string `json:"mitigation"`
}

type oldAIInsightContextWarning struct {
	Field         string `json:"field"`
	UserClaim     string `json:"userClaim"`
	AIExpectation string `json:"aiExpectation"`
	Severity      string `json:"severity"`
	Message       string `json:"message"`
}

type oldAIInsightResp struct {
	SiteAIID         int64                        `json:"site_ai_id"`
	SiteID           string                       `json:"site_id"`
	OverallScore     int                          `json:"overallScore"`
	Recommendation   string                       `json:"recommendation"`
	Headline         string                       `json:"headline"`
	Narrative        string                       `json:"narrative"`
	Metrics          oldAIInsightMetrics          `json:"metrics"`
	Competitors      []oldAIInsightCompetitor     `json:"competitors"`
	Risks            []oldAIInsightRisk           `json:"risks"`
	Opportunities    []string                     `json:"opportunities"`
	ConfidenceLevel  string                       `json:"confidenceLevel"`
	ConfidenceReason string                       `json:"confidenceReason"`
	DataAssumptions  []string                     `json:"dataAssumptions"`
	ContextWarnings  []oldAIInsightContextWarning `json:"contextWarnings"`
	GeneratedAt      time.Time                    `json:"generatedAt"`
}

// ---------------------------------------------------------------------------

var generatedAt = time.Date(2026, 10, 21, 9, 30, 0, 0, time.UTC)

func metric(label string, score int) dto.AIInsightMetric {
	return dto.AIInsightMetric{Label: label, Score: score, Band: "good", Summary: label + " ok", Factors: []string{"a", "b"}}
}

func oldMetric(label string, score int) oldAIInsightMetric {
	return oldAIInsightMetric{Label: label, Score: score, Band: "good", Summary: label + " ok", Factors: []string{"a", "b"}}
}

func geminiInsight() dto.AIInsightResponseGet {
	return dto.AIInsightResponseGet{SiteID: "ST1", OverallScore: 74, Recommendation: "Recommended", Headline: "Toko Kemang — Recommended (74/100)",
		Narrative: "Layak.", Metrics: dto.AIInsightMetrics{FootTraffic: metric("Foot", 80), Demographics: metric("Demo", 70), Competition: metric("Comp", 60),
			Accessibility: metric("Akses", 85), Economic: metric("Eko", 75)},
		Competitors:     []dto.AIInsightCompetitor{{Name: "Indomaret", DistanceKm: 0.05, Type: "Mini Market"}},
		Risks:           []dto.AIInsightRisk{{Title: "Parkir", Severity: "moderate", Mitigation: "sewa lahan"}},
		Opportunities:   []string{"kampus"},
		ConfidenceLevel: "medium", ConfidenceReason: "survey parsial", DataAssumptions: []string{"UMR"},
		ContextWarnings: []dto.AIInsightContextWarning{}, GeneratedAt: generatedAt}
}

func oldGeminiInsight(siteAIID int64) oldAIInsightResp {
	return oldAIInsightResp{SiteAIID: siteAIID, SiteID: "ST1", OverallScore: 74, Recommendation: "Recommended", Headline: "Toko Kemang — Recommended (74/100)",
		Narrative: "Layak.", Metrics: oldAIInsightMetrics{FootTraffic: oldMetric("Foot", 80), Demographics: oldMetric("Demo", 70), Competition: oldMetric("Comp", 60),
			Accessibility: oldMetric("Akses", 85), Economic: oldMetric("Eko", 75)},
		Competitors:     []oldAIInsightCompetitor{{Name: "Indomaret", DistanceKm: 0.05, Type: "Mini Market"}},
		Risks:           []oldAIInsightRisk{{Title: "Parkir", Severity: "moderate", Mitigation: "sewa lahan"}},
		Opportunities:   []string{"kampus"},
		ConfidenceLevel: "medium", ConfidenceReason: "survey parsial", DataAssumptions: []string{"UMR"},
		ContextWarnings: []oldAIInsightContextWarning{}, GeneratedAt: generatedAt}
}

func withGemini(t *testing.T) *mocks.IGeminiClient {
	gemini := mocks.NewIGeminiClient(t)
	withClients(t, func(c *client.Clients) { c.Gemini = gemini })
	return gemini
}

func TestSideBySide_GenerateAIInsight(t *testing.T) {
	gemini := withGemini(t)
	redis := withRedis(t)
	gemini.On("GenerateInsight", mock.Anything, mock.MatchedBy(func(in client.GeminiInput) bool {
		return in.Site.SiteID == "ST1" && in.Context != nil && in.Context.PeakHours == "17-20" && len(in.Competitors) == 2 && len(in.Notes) == 1
	})).Return(geminiInsight(), "gemini-2.5-flash", nil)
	h, rec, fdb := stack(t, append([]sqlfake.Result{{Match: `INSERT INTO "SiteAIInsights"`, Columns: []string{"SiteAIID"}, Rows: [][]driver.Value{{int64(77)}}}},
		detailResults("USER")...)...)

	code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight", map[string]any{"site_id": "ST1"}, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, oldGeminiInsight(77)), body)
	v, _ := redis.Get(quotaKey("ai_insight", 1001))
	assert.Equal(t, "49", v, "one of 50 used")
	sqls := stableSQL(rec.sqls)
	assert.Equal(t, []string{
		`SELECT * FROM "SystemSettings" WHERE SettingKey = 'ai_insight.daily_quota' ORDER BY "SystemSettings"."ID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		oldDetailSQL[2], oldDetailSQL[5], oldDetailSQL[6], oldDetailSQL[7],
		`DELETE FROM "SiteAIInsights" WHERE SiteID = 'ST1'`,
		// DIFFERENCE (Wes, 27 Oct): 2 new columns, NULL for an insight
		// without them (this mock answer has the old shape)
		`INSERT INTO "SiteAIInsights" ("SiteID","OverallScore","Recommendation","Headline","Narrative","ConfidenceLevel","ConfidenceReason","GeminiModel","GeneratedBy","GeneratedAt","CreatedAt","ScoreReason","ScoreBreakdown") OUTPUT INSERTED."SiteAIID" VALUES ('ST1',74,'Recommended','Toko Kemang — Recommended (74/100)','Layak.','medium','survey parsial','gemini-2.5-flash',1001,'<time>','<time>',NULL,NULL);`,
		`INSERT INTO "SiteAIMetrics" ("SiteAIID","MetricKey","Label","Score","Band","Summary","Factors","CreatedAt") OUTPUT INSERTED."ID" VALUES (77,'footTraffic','Foot',80,'good','Foot ok','a;b','<time>'),(77,'demographics','Demo',70,'good','Demo ok','a;b','<time>'),(77,'competition','Comp',60,'good','Comp ok','a;b','<time>'),(77,'accessibility','Akses',85,'good','Akses ok','a;b','<time>'),(77,'economic','Eko',75,'good','Eko ok','a;b','<time>');`,
		`INSERT INTO "SiteAIRisks" ("SiteAIID","Title","Severity","Mitigation","CreatedAt") OUTPUT INSERTED."ID" VALUES (77,'Parkir','moderate','sewa lahan','<time>');`,
		`INSERT INTO "SiteAIOpportunities" ("SiteAIID","Opportunity","CreatedAt") OUTPUT INSERTED."ID" VALUES (77,'kampus','<time>');`,
		`INSERT INTO "SiteAIDataAssumptions" ("SiteAIID","Assumption","CreatedAt") OUTPUT INSERTED."ID" VALUES (77,'UMR','<time>');`,
	}, sqls)
	assert.Contains(t, fdb.Queries(), "COMMIT")
}

func TestSideBySide_GenerateAIInsight_QuotaUsedUp(t *testing.T) {
	gemini := withGemini(t)
	redis := withRedis(t)
	redis.Set(quotaKey("ai_insight", 1001), "0")
	h, _, _ := stack(t, detailResults("USER")...)

	code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight", map[string]any{"site_id": "ST1"}, 1001)

	assert.Equal(t, http.StatusTooManyRequests, code)
	assertSameJSON(t, []byte(`{"code":429,"status":"failure","message":"Too Many Requests","errors":{"error":"Kuota generate AI insight harian (50) sudah habis, coba lagi besok."}}`), body)
	gemini.AssertNotCalled(t, "GenerateInsight")
}

// No gemini.apikey: the real client answers ErrGeminiNotSetUp, the API
// answers 503 "AI not available" (the app then sends without AI score).
func TestGenerateAIInsight_GeminiKeyNotSet(t *testing.T) {
	withRedis(t)
	h, _, _ := stack(t, detailResults("USER")...)

	code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight", map[string]any{"site_id": "ST1"}, 1001)

	assert.Equal(t, http.StatusServiceUnavailable, code)
	assertSameJSON(t, []byte(`{"code":503,"status":"failure","message":"Service Unavailable","errors":{"error":"Analisis AI belum aktif di server ini (kunci Gemini belum diisi, gemini.apikey)."}}`), body)
}

func TestSideBySide_AIInsightQuota(t *testing.T) {
	redis := withRedis(t)
	redis.Set(quotaKey("ai_insight", 1001), "12")
	h, _, _ := stack(t)

	code, body := call(t, h, http.MethodGet, "/v1/task/ai-insight/quota", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, oldAIInsightQuotaResp{Limit: 50, Remaining: 12}), body)
}

func TestSideBySide_GetAIInsight(t *testing.T) {
	metricCols := []string{"ID", "SiteAIID", "MetricKey", "Label", "Score", "Band", "Summary", "Factors"}
	h, rec, _ := stack(t, append([]sqlfake.Result{
		{Match: `FROM "SiteAIInsights"`, Columns: []string{"SiteAIID", "SiteID", "OverallScore", "Recommendation", "Headline", "Narrative", "ConfidenceLevel", "ConfidenceReason", "GeminiModel", "GeneratedBy", "GeneratedAt"},
			Rows: [][]driver.Value{{int64(77), "ST1", int64(74), "Recommended", "Toko Kemang — Recommended (74/100)", "Layak.", "medium", "survey parsial", "gemini-2.5-flash", int64(1001), generatedAt}}},
		{Match: `FROM "SiteAIMetrics"`, Columns: metricCols, Rows: [][]driver.Value{
			{int64(1), int64(77), "footTraffic", "Foot", int64(80), "good", "Foot ok", "a;b"},
			{int64(2), int64(77), "demographics", "Demo", int64(70), "good", "Demo ok", "a;b"},
			{int64(3), int64(77), "competition", "Comp", int64(60), "good", "Comp ok", "a;b"},
			{int64(4), int64(77), "accessibility", "Akses", int64(85), "good", "Akses ok", "a;b"},
			{int64(5), int64(77), "economic", "Eko", int64(75), "good", "Eko ok", "a;b"},
		}},
		{Match: `FROM "SiteAIRisks"`, Columns: []string{"ID", "SiteAIID", "Title", "Severity", "Mitigation"}, Rows: [][]driver.Value{{int64(1), int64(77), "Parkir", "moderate", "sewa lahan"}}},
		{Match: `FROM "SiteAIOpportunities"`, Columns: []string{"ID", "SiteAIID", "Opportunity"}, Rows: [][]driver.Value{{int64(1), int64(77), "kampus"}}},
		{Match: `FROM "SiteAIDataAssumptions"`, Columns: []string{"ID", "SiteAIID", "Assumption"}, Rows: [][]driver.Value{{int64(1), int64(77), "UMR"}}},
	}, detailResults("USER")...)...)

	code, body := call(t, h, http.MethodGet, "/v1/task/ai-insight/ST1", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	want := oldGeminiInsight(77)
	// competitors come from the survey (SiteCompetitors), distance read as km
	want.Competitors = []oldAIInsightCompetitor{{Name: "Indomaret", DistanceKm: 0.05, Type: "minimarket"}, {Name: "Alfamart", DistanceKm: 0.12, Type: "minimarket"}}
	assertSameJSON(t, oldDataEnvelope(t, want), body)
	assert.Equal(t, []string{
		`SELECT * FROM "SiteAIInsights" WHERE SiteID = 'ST1' ORDER BY GeneratedAt DESC,"SiteAIInsights"."SiteAIID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "SiteAIMetrics" WHERE SiteAIID = 77 ORDER BY ID ASC`,
		`SELECT * FROM "SiteAIRisks" WHERE SiteAIID = 77 ORDER BY ID ASC`,
		`SELECT * FROM "SiteAIOpportunities" WHERE SiteAIID = 77 ORDER BY ID ASC`,
		`SELECT * FROM "SiteAIDataAssumptions" WHERE SiteAIID = 77 ORDER BY ID ASC`,
		`SELECT * FROM "SiteAIContextWarnings" WHERE SiteAIID = 77 ORDER BY ID ASC`,
		oldDetailSQL[6],
	}, rec.sqls)

	h, _, _ = stack(t)
	code, body = call(t, h, http.MethodGet, "/v1/task/ai-insight/ST9", nil, 1001)
	assert.Equal(t, http.StatusInternalServerError, code)
	assertSameJSON(t, []byte(`{"code":500,"status":"failure","message":"Internal Server Error","errors":{"error":"insight for site ST9: record not found"}}`), body)
}

func TestSideBySide_PreviewAIInsight(t *testing.T) {
	gemini := withGemini(t)
	withRedis(t)
	insight := geminiInsight()
	insight.SiteID = ""
	gemini.On("GenerateInsight", mock.Anything, mock.MatchedBy(func(in client.GeminiInput) bool {
		return in.Site.SiteID == "" && in.Site.SiteLat == -6.2 && in.Context != nil && *in.Context.ParkingAvailability == "LIMITED" && len(in.Competitors) == 1
	})).Return(insight, "gemini-2.5-flash", nil)
	h, rec, _ := stack(t)

	code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", map[string]any{
		"site":    map[string]any{"site_name_1": "Baru", "site_lat": -6.2, "site_lon": 106.8, "site_planned_opening_date": "2027-01-15"},
		"context": map[string]any{"parking_availability": "LIMITED", "competitors": []map[string]any{{"competitor_name": "Alfamart"}}},
	}, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	want := oldGeminiInsight(0)
	want.SiteID = ""
	assertSameJSON(t, oldDataEnvelope(t, want), body)
	for _, sql := range rec.sqls {
		assert.False(t, strings.HasPrefix(sql, "INSERT") || strings.HasPrefix(sql, "DELETE"), "preview stores nothing: %s", sql)
	}

	code, _ = call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", map[string]any{"site": map[string]any{"site_name_1": "x"}}, 1001)
	assert.Equal(t, http.StatusUnprocessableEntity, code) // DIFFERENCE: site-dev-api 500 (go-playground required)
}

// Gemini failures answer the "AI not available" codes the app knows (5xx:
// send without AI score). DIFFERENCE (fix, 23 Oct): the quota is not used
// (site-dev-api used one per failed try).
func TestAIInsight_GeminiFailureAnswers(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code int
		want string
	}{
		{&client.GeminiDeniedError{HTTPStatus: 400, Code: "API_KEY_INVALID", Reason: "kunci Gemini tidak valid", Message: "API key not valid."}, http.StatusServiceUnavailable,
			`{"code":503,"status":"failure","message":"Service Unavailable","errors":{"error":"Analisis AI tidak tersedia: kunci Gemini tidak valid."}}`},
		{&client.GeminiUpstreamError{Timeout: true, Detail: "no answer within 1m0s"}, http.StatusGatewayTimeout,
			`{"code":504,"status":"failure","message":"Gateway Timeout","errors":{"error":"Analisis AI tidak tersedia: Gemini tidak menjawab tepat waktu. Coba lagi sebentar lagi."}}`},
		{&client.GeminiUpstreamError{Detail: "HTTP 503 UNAVAILABLE"}, http.StatusBadGateway,
			`{"code":502,"status":"failure","message":"Bad Gateway","errors":{"error":"Analisis AI tidak tersedia sekarang (Gemini gagal menjawab). Coba lagi sebentar lagi."}}`},
	} {
		gemini := withGemini(t)
		redis := withRedis(t)
		gemini.On("GenerateInsight", mock.Anything, mock.Anything).Return(dto.AIInsightResponseGet{}, "", tc.err)
		h, _, _ := stack(t)

		code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", previewAt(-6.2, 106.8), 1001)

		assert.Equal(t, tc.code, code)
		assertSameJSON(t, []byte(tc.want), body)
		left, _ := redis.Get(quotaKey("ai_insight", 1001))
		assert.Equal(t, "50", left, "a failed try is given back")
	}
}

// Preview passes the November survey fields to the prompt (traffic, parking
// and competitor notes), like a saved survey.
func TestPreviewAIInsight_TrafficFieldsReachGemini(t *testing.T) {
	gemini := withGemini(t)
	withRedis(t)
	gemini.On("GenerateInsight", mock.Anything, mock.MatchedBy(func(in client.GeminiInput) bool {
		c := in.Context
		return c != nil && c.VehicleTrafficLevel != nil && *c.VehicleTrafficLevel == "BUSY" && c.VehicleTrafficCount != nil && *c.VehicleTrafficCount == 40 &&
			*c.VehicleTrafficMinutes == 10 && *c.VehicleTrafficNote == "macet sore" && *c.ParkingNote == "parkir motor" && *c.CompetitorNote == "Alfamart renovasi"
	})).Return(geminiInsight(), "gemini-2.5-flash", nil)
	h, _, _ := stack(t)

	code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", map[string]any{
		"site": map[string]any{"site_name_1": "Baru", "site_lat": -6.2, "site_lon": 106.8, "site_planned_opening_date": "2027-01-15"},
		"context": map[string]any{"vehicle_traffic_level": "BUSY", "vehicle_traffic_count": 40, "vehicle_traffic_minutes": 10, "vehicle_traffic_note": "macet sore",
			"parking_note": "parkir motor", "competitor_note": "Alfamart renovasi"},
	}, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
}
