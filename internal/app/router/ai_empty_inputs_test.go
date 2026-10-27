package router

// No AI on empty survey inputs (Wes, 27 Oct): name + pin + sales area alone
// are refused with 400 and the list of what to fill; nothing is counted and
// Gemini is never called. One filled survey input is enough.

import (
	"database/sql/driver"
	"net/http"
	"testing"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const emptyInputsAnswer = `{"code":400,"status":"failure","message":"Bad Request","errors":{"error":"Analisis AI belum bisa dibuat: data survei masih kosong. Isi minimal satu: keramaian orang lewat, jam ramai, lalu lintas kendaraan, parkir, transportasi umum, landmark sekitar, penduduk sekitar, catatan kompetitor, atau catatan tambahan."}}`

func TestAIEmptyInputs_PreviewRefused(t *testing.T) {
	for name, ctx := range map[string]any{
		"no context":           nil,
		"empty context":        map[string]any{},
		"blanks only":          map[string]any{"peak_hours": "  ", "parking_note": " ", "custom_notes": []map[string]any{{"title": " ", "content": ""}}},
		"map competitors only": map[string]any{"competitors": []map[string]any{{"competitor_name": "Indomaret", "distance": "50m"}}},
	} {
		t.Run(name, func(t *testing.T) {
			gemini := withGemini(t)
			redis := withRedis(t)
			limitsOn(t, 5, 60)
			h, _, _ := stack(t)
			body := map[string]any{"site": map[string]any{"site_name_1": "Baru", "site_lat": -6.2, "site_lon": 106.8, "site_sales_area": 120}}
			if ctx != nil {
				body["context"] = ctx
			}

			code, got := call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", body, 1001)

			assert.Equal(t, http.StatusBadRequest, code)
			assertSameJSON(t, []byte(emptyInputsAnswer), got)
			gemini.AssertNotCalled(t, "GenerateInsight")
			v, _ := redis.Get(quotaKey("ai_insight", 1001))
			assert.Empty(t, v, "nothing counted")
		})
	}
}

func TestAIEmptyInputs_AnyOneFieldIsEnough(t *testing.T) {
	for name, ctx := range map[string]any{
		"foot traffic":    map[string]any{"foot_traffic_observation": "ramai pagi"},
		"traffic level":   map[string]any{"vehicle_traffic_level": "BUSY"},
		"traffic count":   map[string]any{"vehicle_traffic_count": 0, "vehicle_traffic_minutes": 10},
		"parking":         map[string]any{"parking_availability": "NONE"},
		"parking note":    map[string]any{"parking_note": "motor saja"},
		"transport":       map[string]any{"public_transport": "halte 50 m"},
		"landmark":        map[string]any{"anchor_landmark": "masjid"},
		"demographics":    map[string]any{"observed_demographics": "mahasiswa"},
		"competitor note": map[string]any{"competitor_note": "Alfamart tutup"},
		"custom note":     map[string]any{"custom_notes": []map[string]any{{"title": "Banjir", "content": "tidak pernah"}}},
	} {
		t.Run(name, func(t *testing.T) {
			gemini := withGemini(t)
			withRedis(t)
			gemini.On("GenerateInsight", mock.Anything, mock.Anything).Return(geminiInsight(), "gemini-2.5-flash", nil)
			h, _, _ := stack(t)

			code, got := call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", map[string]any{
				"site": map[string]any{"site_name_1": "Baru", "site_lat": -6.2, "site_lon": 106.8}, "context": ctx}, 1001)

			assert.Equal(t, http.StatusOK, code, string(got))
		})
	}
}

// A saved site without any survey: generate refused, quota given back.
func TestAIEmptyInputs_GenerateRefused(t *testing.T) {
	gemini := withGemini(t)
	redis := withRedis(t)
	results := []sqlfake.Result{noRows(`FROM "SiteContexts"`)}
	h, _, _ := stack(t, append(results, detailResults("USER")...)...)

	code, got := call(t, h, http.MethodPost, "/v1/task/ai-insight", map[string]any{"site_id": "ST1"}, 1001)

	assert.Equal(t, http.StatusBadRequest, code)
	assertSameJSON(t, []byte(emptyInputsAnswer), got)
	gemini.AssertNotCalled(t, "GenerateInsight")
	left, _ := redis.Get(quotaKey("ai_insight", 1001))
	assert.Equal(t, "50", left, "the held try is given back")
}

// The queued job ends with 200 (no retry) when there is nothing to analyse.
func TestAIEmptyInputs_QueueJobNotRetried(t *testing.T) {
	m := withFlow(t)
	jwtServer(t, http.StatusOK, `{}`) // sets the test public_api username/password
	h, _, _ := stack(t, noRows(`FROM "SiteContexts"`),
		sqlfake.Result{Match: `FROM "Sites" WHERE SiteID = @p1`, Columns: siteColumns, Rows: [][]driver.Value{siteST1Row()}})

	code, body := basicCall(t, h, "/v1/queue/ai-insight", "test-user", "test-pass", `{"site_id":"ST1","actor_id":1001}`)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success"}`), body)
	m.gemini.AssertNotCalled(t, "GenerateInsight")
}
