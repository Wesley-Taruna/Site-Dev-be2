package router

// AI analysis limits (Wes, 23 Oct): [ai_insight] site_quota (per site
// location, all users, all rounds), daily_cap (all users per day), on top of
// the per-user daily_quota. Only answered Gemini calls count. Gemini is a
// mock (no live calls).

import (
	"net/http"
	"testing"

	"site-dev-backend/internal/app/client"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// limitsOn sets [ai_insight] site_quota / daily_cap for the stacks built
// after it (0 = off).
func limitsOn(t *testing.T, siteQuota, dailyCap int) {
	testEnv.Set("ai_insight.site_quota", siteQuota)
	testEnv.Set("ai_insight.daily_cap", dailyCap)
	t.Cleanup(func() {
		testEnv.Set("ai_insight.site_quota", 0)
		testEnv.Set("ai_insight.daily_cap", 0)
	})
}

func previewAt(lat, lng float64) map[string]any {
	// one survey input: no AI on an empty survey (Wes, 27 Oct)
	return map[string]any{"site": map[string]any{"site_name_1": "Baru", "site_lat": lat, "site_lon": lng, "site_planned_opening_date": "2027-01-15"},
		"context": map[string]any{"peak_hours": "17-20"}}
}

func geminiAnswers(t *testing.T) *mocks.IGeminiClient {
	gemini := withGemini(t)
	gemini.On("GenerateInsight", mock.Anything, mock.Anything).Return(geminiInsight(), "gemini-2.5-flash", nil)
	return gemini
}

func TestAILimits_PerSite_ThirdOKFourthRefused(t *testing.T) {
	limitsOn(t, 3, 0)
	gemini := geminiAnswers(t)
	redis := withRedis(t)
	h, _, _ := stack(t, detailResults("USER")...)

	// 3 tries at ST1's spot (-6.26, 106.81), one by another user, pin moved
	// a few metres: one site
	for i, p := range [][2]float64{{-6.26, 106.81}, {-6.2601, 106.8102}, {-6.2599, 106.8098}} {
		user := int64(1001)
		if i == 1 {
			user = 3001
		}
		code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", previewAt(p[0], p[1]), user)
		require.Equal(t, http.StatusOK, code, "try %d: %s", i+1, body)
	}

	code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", previewAt(-6.26, 106.81), 1001)
	assert.Equal(t, http.StatusTooManyRequests, code)
	assertSameJSON(t, []byte(`{"code":429,"status":"failure","message":"Too Many Requests","errors":{"error":"Batas analisis AI untuk lokasi ini sudah tercapai (3x)."}}`), body)
	gemini.AssertNumberOfCalls(t, "GenerateInsight", 3)
	left, _ := redis.Get(quotaKey("ai_insight", 1001))
	assert.Equal(t, "48", left, "2 answered tries; the refused 4th is given back")

	// the saved site ST1 (same spot, generate, another staff) counts with them
	code, body = call(t, h, http.MethodPost, "/v1/task/ai-insight", map[string]any{"site_id": "ST1"}, 2002)
	assert.Equal(t, http.StatusTooManyRequests, code)
	assertSameJSON(t, []byte(`{"code":429,"status":"failure","message":"Too Many Requests","errors":{"error":"Batas analisis AI untuk lokasi ini sudah tercapai (3x)."}}`), body)
	left, _ = redis.Get(quotaKey("ai_insight", 2002))
	assert.Equal(t, "50", left)

	// another spot (~1 km away) is another site
	code, _ = call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", previewAt(-6.27, 106.81), 1001)
	assert.Equal(t, http.StatusOK, code)
	gemini.AssertNumberOfCalls(t, "GenerateInsight", 4)
}

func TestAILimits_ServerWideDailyCap(t *testing.T) {
	limitsOn(t, 0, 2)
	gemini := geminiAnswers(t)
	withRedis(t)
	h, _, _ := stack(t)

	for i, user := range []int64{1001, 2002} {
		code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", previewAt(-6.2+float64(i)/10, 106.8), user)
		require.Equal(t, http.StatusOK, code, string(body))
	}
	code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", previewAt(-6.5, 106.8), 3001)

	assert.Equal(t, http.StatusTooManyRequests, code)
	assertSameJSON(t, []byte(`{"code":429,"status":"failure","message":"Too Many Requests","errors":{"error":"Batas analisis AI hari ini untuk semua pengguna sudah tercapai (2x), coba lagi besok."}}`), body)
	gemini.AssertNumberOfCalls(t, "GenerateInsight", 2)

	code, body = call(t, h, http.MethodGet, "/v1/task/ai-insight/quota", nil, 3001)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, dto.AIInsightQuotaResponse{Limit: 50, Remaining: 50, ServerToday: &dto.AIInsightLimitCount{Limit: 2, Used: 2, Remaining: 0}}), body)
}

// Per user per day = the existing daily_quota (env.toml, or the admin's
// SystemSettings value): message unchanged; a refused try touches no other
// count.
func TestAILimits_PerUserDaily(t *testing.T) {
	limitsOn(t, 3, 60)
	testEnv.Set("ai_insight.daily_quota", 10)
	t.Cleanup(func() { testEnv.Set("ai_insight.daily_quota", 0) })
	gemini := geminiAnswers(t)
	redis := withRedis(t)
	redis.Set(quotaKey("ai_insight", 1001), "1")
	h, _, _ := stack(t)

	code, _ := call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", previewAt(-6.26, 106.81), 1001)
	require.Equal(t, http.StatusOK, code)
	code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", previewAt(-6.26, 106.81), 1001)

	assert.Equal(t, http.StatusTooManyRequests, code)
	assertSameJSON(t, []byte(`{"code":429,"status":"failure","message":"Too Many Requests","errors":{"error":"Kuota generate AI insight harian (10) sudah habis, coba lagi besok."}}`), body)
	gemini.AssertNumberOfCalls(t, "GenerateInsight", 1)
	code, body = call(t, h, http.MethodGet, "/v1/task/ai-insight/quota?lat=-6.26&lng=106.81", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, dto.AIInsightQuotaResponse{Limit: 10, Remaining: 0,
		Site:        &dto.AIInsightLimitCount{Limit: 3, Used: 1, Remaining: 2},
		ServerToday: &dto.AIInsightLimitCount{Limit: 60, Used: 1, Remaining: 59}}), body)
}

// A failed Gemini call (old bug: quota used anyway) counts nowhere.
func TestAILimits_FailedGeminiNotCounted(t *testing.T) {
	limitsOn(t, 3, 60)
	gemini := withGemini(t)
	redis := withRedis(t)
	gemini.On("GenerateInsight", mock.Anything, mock.Anything).Return(dto.AIInsightResponseGet{}, "", &client.GeminiUpstreamError{Detail: "HTTP 503 UNAVAILABLE"})
	h, _, _ := stack(t, detailResults("USER")...)

	code, _ := call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", previewAt(-6.26, 106.81), 1001)
	assert.Equal(t, http.StatusBadGateway, code)
	code, _ = call(t, h, http.MethodPost, "/v1/task/ai-insight", map[string]any{"site_id": "ST1"}, 1001)
	assert.Equal(t, http.StatusBadGateway, code)

	left, _ := redis.Get(quotaKey("ai_insight", 1001))
	assert.Equal(t, "50", left)
	code, body := call(t, h, http.MethodGet, "/v1/task/ai-insight/quota?site_id=ST1", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, dto.AIInsightQuotaResponse{Limit: 50, Remaining: 50,
		Site:        &dto.AIInsightLimitCount{Limit: 3, Used: 0, Remaining: 3},
		ServerToday: &dto.AIInsightLimitCount{Limit: 60, Used: 0, Remaining: 60}}), body)
}

// Default (limits off, main): nothing new is counted or answered, even when
// a location is asked.
func TestAILimits_OffByDefault(t *testing.T) {
	gemini := geminiAnswers(t)
	redis := withRedis(t)
	h, _, _ := stack(t)

	for i := 0; i < 4; i++ {
		code, _ := call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", previewAt(-6.26, 106.81), 1001)
		require.Equal(t, http.StatusOK, code)
	}
	gemini.AssertNumberOfCalls(t, "GenerateInsight", 4)
	assert.Len(t, redis.Keys(), 1, "only the per-user counter")

	code, body := call(t, h, http.MethodGet, "/v1/task/ai-insight/quota?lat=-6.26&lng=106.81", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, oldAIInsightQuotaResp{Limit: 50, Remaining: 46}), body)

	code, _ = call(t, h, http.MethodGet, "/v1/task/ai-insight/quota?lat=abc", nil, 1001)
	assert.Equal(t, http.StatusUnprocessableEntity, code)
}
