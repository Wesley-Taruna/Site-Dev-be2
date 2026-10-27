package router

// Re-run after a revision / on a reopened draft (Wes, 27 Oct): a new AI
// result only when the inputs changed. Same inputs as the newest stored
// insight = 409 "Data belum berubah sejak analisis terakhir.", no Gemini
// call, nothing counted. Earlier insights are kept (no DELETE).

import (
	"database/sql/driver"
	"net/http"
	"strings"
	"testing"

	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const unchangedAnswer = `{"code":409,"status":"failure","message":"Conflict","errors":{"error":"Data belum berubah sejak analisis terakhir. Ubah data survei atau lokasi dulu untuk membuat analisis baru."}}`

// latestInsight answers the "newest insight of ST1" read with inputHash.
func latestInsight(inputHash string) sqlfake.Result {
	return sqlfake.Result{Match: `FROM "SiteAIInsights" WHERE SiteID = @p1 ORDER BY SiteAIID DESC`,
		Columns: []string{"SiteAIID", "SiteID", "OverallScore", "InputHash"},
		Rows:    [][]driver.Value{{int64(77), "ST1", int64(70), inputHash}}}
}

func insertedInsight() sqlfake.Result {
	return sqlfake.Result{Match: `INSERT INTO "SiteAIInsights"`, Columns: []string{"SiteAIID"}, Rows: [][]driver.Value{{int64(78)}}}
}

// hashOfGenerate is the input_hash of a first generate on ST1's saved data.
func hashOfGenerate(t *testing.T) string {
	gemini := withGemini(t)
	withRedis(t)
	gemini.On("GenerateInsight", mock.Anything, mock.Anything).Return(geminiInsight(), "gemini-2.5-flash", nil).Once()
	h, _, _ := stack(t, append([]sqlfake.Result{insertedInsight()}, detailResults("USER")...)...)
	code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight", map[string]any{"site_id": "ST1"}, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	var got dto.AIInsightResponseGet
	require.NoError(t, jsonUnmarshalData(body, &got))
	require.Len(t, got.InputHash, 64)
	return got.InputHash
}

func TestAIRerun_Generate_SameInputsRefused(t *testing.T) {
	hash := hashOfGenerate(t)
	gemini := withGemini(t)
	redis := withRedis(t)
	h, rec, _ := stack(t, append([]sqlfake.Result{latestInsight(hash)}, detailResults("USER")...)...)

	code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight", map[string]any{"site_id": "ST1"}, 1001)

	assert.Equal(t, http.StatusConflict, code)
	assertSameJSON(t, []byte(unchangedAnswer), body)
	gemini.AssertNotCalled(t, "GenerateInsight")
	left, _ := redis.Get(quotaKey("ai_insight", 1001))
	assert.Equal(t, "50", left, "nothing counted")
	for _, s := range rec.sqls {
		assert.False(t, strings.HasPrefix(s, "INSERT"), s)
	}
}

func TestAIRerun_Generate_ChangedInputsNewResultKeepsHistory(t *testing.T) {
	gemini := withGemini(t)
	redis := withRedis(t)
	gemini.On("GenerateInsight", mock.Anything, mock.Anything).Return(scoredGeminiInsight(), "gemini-2.5-flash", nil)
	h, rec, _ := stack(t, append([]sqlfake.Result{latestInsight("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"), insertedInsight()},
		detailResults("USER")...)...)

	code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight", map[string]any{"site_id": "ST1"}, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Contains(t, string(body), `"site_ai_id":78`)
	left, _ := redis.Get(quotaKey("ai_insight", 1001))
	assert.Equal(t, "49", left)
	joined := strings.Join(rec.sqls, "\n")
	assert.NotContains(t, joined, `DELETE FROM "SiteAIInsights"`, "the earlier result stays as history")
	assert.Contains(t, joined, `INSERT INTO "SiteAIInsights"`)
}

// Old insights (no fingerprint) never block a re-run.
func TestAIRerun_Generate_OldInsightWithoutHash(t *testing.T) {
	gemini := withGemini(t)
	withRedis(t)
	gemini.On("GenerateInsight", mock.Anything, mock.Anything).Return(geminiInsight(), "gemini-2.5-flash", nil)
	old := sqlfake.Result{Match: `FROM "SiteAIInsights" WHERE SiteID = @p1 ORDER BY SiteAIID DESC`,
		Columns: []string{"SiteAIID", "SiteID", "OverallScore", "InputHash"}, Rows: [][]driver.Value{{int64(77), "ST1", int64(70), nil}}}
	h, _, _ := stack(t, append([]sqlfake.Result{old, insertedInsight()}, detailResults("USER")...)...)

	code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight", map[string]any{"site_id": "ST1"}, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
}

func previewBody(siteID string, peak string) map[string]any {
	b := map[string]any{
		"site":    map[string]any{"site_name_1": "Toko Kemang", "site_lat": -6.26, "site_lon": 106.81, "site_sales_area": 120},
		"context": map[string]any{"peak_hours": peak, "parking_availability": "LIMITED"},
	}
	if siteID != "" {
		b["site_id"] = siteID
	}
	return b
}

func TestAIRerun_Preview(t *testing.T) {
	gemini := withGemini(t)
	withRedis(t)
	gemini.On("GenerateInsight", mock.Anything, mock.Anything).Return(geminiInsight(), "gemini-2.5-flash", nil)
	h, _, _ := stack(t)
	code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", previewBody("", "17-20"), 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	var first dto.AIInsightResponseGet
	require.NoError(t, jsonUnmarshalData(body, &first))

	// the saved request (twin ID in the body works too) whose newest insight is that preview
	for _, id := range []string{"ST1", "SD27-10-2026001"} {
		gemini := withGemini(t)
		withRedis(t)
		match := sqlfake.Result{Match: `FROM "SiteAIInsights" WHERE SiteID = @p1 ORDER BY SiteAIID DESC`,
			Columns: []string{"SiteAIID", "SiteID", "InputHash"}, Rows: [][]driver.Value{{int64(77), "x", first.InputHash}}}
		h, rec, _ := stack(t, match)

		code, body = call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", previewBody(id, "17-20"), 1001)
		assert.Equal(t, http.StatusConflict, code, id)
		assertSameJSON(t, []byte(unchangedAnswer), body)
		gemini.AssertNotCalled(t, "GenerateInsight")
		if id != "ST1" {
			assert.True(t, sqlMentions(rec.sqls, "'"+newSiteID+"'"))
		}

		// one input changed: a new result
		gemini.On("GenerateInsight", mock.Anything, mock.Anything).Return(geminiInsight(), "gemini-2.5-flash", nil)
		code, body = call(t, h, http.MethodPost, "/v1/task/ai-insight/preview", previewBody(id, "17-21"), 1001)
		assert.Equal(t, http.StatusOK, code, string(body))
	}
}

func TestAIRerun_QueueJobUnchangedNotRetried(t *testing.T) {
	hash := hashOfGenerate(t)
	m := withFlow(t)
	jwtServer(t, http.StatusOK, `{}`)
	h, _, _ := stack(t, append([]sqlfake.Result{latestInsight(hash)}, detailResults("USER")...)...)

	code, body := basicCall(t, h, "/v1/queue/ai-insight", "test-user", "test-pass", `{"site_id":"ST1","actor_id":1001}`)

	require.Equal(t, http.StatusOK, code, string(body))
	m.gemini.AssertNotCalled(t, "GenerateInsight")
}
