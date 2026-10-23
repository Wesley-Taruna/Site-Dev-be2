package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testGeminiKey = "TEST-GEMINI-KEY-not-a-real-key"

// fakeGemini answers generateContent from testdata fixtures (recorded
// shapes, no live call): one answer per try, the last one repeated.
type fakeGemini struct {
	t       *testing.T
	mu      sync.Mutex
	answers []fakeAnswer
	calls   []geminiCall
	netErr  error
}

type fakeAnswer struct {
	status  int
	fixture string
}

type geminiCall struct {
	url  string
	key  string
	body map[string]any
}

func (f *fakeGemini) RoundTrip(r *http.Request) (*http.Response, error) {
	raw, _ := io.ReadAll(r.Body)
	c := geminiCall{url: r.URL.String(), key: r.Header.Get("x-goog-api-key")}
	require.NoError(f.t, json.Unmarshal(raw, &c.body))
	f.mu.Lock()
	f.calls = append(f.calls, c)
	n := len(f.calls)
	f.mu.Unlock()
	if f.netErr != nil {
		return nil, f.netErr
	}
	a := f.answers[len(f.answers)-1]
	if n <= len(f.answers) {
		a = f.answers[n-1]
	}
	return answer(a.status, fixture(f.t, a.fixture))
}

func geminiClient(t *testing.T, answers ...fakeAnswer) (*GeminiClient, *fakeGemini) {
	f := &fakeGemini{t: t, answers: answers}
	return &GeminiClient{
		apiKey: testGeminiKey, model: "gemini-2.5-flash", baseURL: "https://gemini.example.test/v1beta/models",
		http: &http.Client{Transport: f, Timeout: 5 * time.Second}, retryWait: time.Millisecond,
		now: func() time.Time { return time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC) },
	}, f
}

func geminiTestInput() GeminiInput {
	site, ctx, comps, notes := promptFixtureSite()
	return GeminiInput{Site: site, Context: ctx, Competitors: comps, Notes: notes}
}

// assertNoKey: the key never reaches an error text or a URL.
func assertNoKey(t *testing.T, err error, f *fakeGemini) {
	t.Helper()
	if err != nil {
		assert.NotContains(t, err.Error(), testGeminiKey)
	}
	for _, c := range f.calls {
		assert.NotContains(t, c.url, testGeminiKey)
		assert.NotContains(t, c.url, "key=")
	}
}

func TestGemini_OK_ScoreCardWithSixMetrics(t *testing.T) {
	c, f := geminiClient(t, fakeAnswer{200, "gemini_ok.json"})
	in := geminiTestInput()
	var usage GeminiUsage

	got, model, err := c.GenerateInsight(WithGeminiUsage(context.Background(), &usage), in)

	require.NoError(t, err)
	assert.Equal(t, "gemini-2.5-flash", model)
	require.Len(t, f.calls, 1)
	assert.Equal(t, "https://gemini.example.test/v1beta/models/gemini-2.5-flash:generateContent", f.calls[0].url)
	assert.Equal(t, testGeminiKey, f.calls[0].key, "key only in the x-goog-api-key header")
	assertNoKey(t, err, f)

	// the body is exactly GeminiRequest(in): prompt + schema + config
	want, err := GeminiRequest(in)
	require.NoError(t, err)
	var wantBody map[string]any
	require.NoError(t, json.Unmarshal(want, &wantBody))
	assert.Equal(t, wantBody, f.calls[0].body)

	// the answer parses exactly as the services expect
	assert.Equal(t, in.Site.SiteID, got.SiteID)
	assert.Equal(t, 72, got.OverallScore)
	assert.Equal(t, "Recommended", got.Recommendation)
	assert.Equal(t, "Kemang Timur — Recommended (72/100)", got.Headline, "built from the site name, not Gemini's text")
	assert.Equal(t, 75, got.Metrics.FootTraffic.Score)
	assert.Equal(t, 70, got.Metrics.Demographics.Score)
	assert.Equal(t, 60, got.Metrics.Competition.Score)
	assert.Equal(t, 78, got.Metrics.Accessibility.Score)
	assert.Equal(t, 72, got.Metrics.Economic.Score)
	require.NotNil(t, got.Metrics.VehicleTraffic, "6th metric vehicleTraffic")
	assert.Equal(t, 80, got.Metrics.VehicleTraffic.Score)
	assert.Equal(t, "Lalu Lintas Kendaraan", got.Metrics.VehicleTraffic.Label)
	assert.Equal(t, []string{"faktor 1", "faktor 2"}, got.Metrics.Economic.Factors)
	assert.Len(t, got.Competitors, 2)
	assert.Equal(t, 0.18, got.Competitors[0].DistanceKm)
	assert.Len(t, got.Risks, 1)
	assert.Len(t, got.Opportunities, 3)
	assert.Equal(t, "medium", got.ConfidenceLevel)
	assert.Len(t, got.ContextWarnings, 1)
	assert.Equal(t, time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC), got.GeneratedAt)

	assert.Equal(t, GeminiUsage{Model: "gemini-2.5-flash", PromptTokens: 3412, OutputTokens: 1650, ThoughtsTokens: 2000, TotalTokens: 7062, Attempts: 1}, usage)
}

func TestParseGeminiAnswer_HeadlineBuiltHere(t *testing.T) {
	inner := `{"overallScore":81,"recommendation":"Recommended","headline":"Ruko \\u2014 Recommended (81/100)"}`
	raw, err := json.Marshal(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{map[string]any{"text": inner}}}}}})
	require.NoError(t, err)

	got, err := ParseGeminiAnswer(raw, &model.Site{SiteName1: "Ruko Tebet"}, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "Ruko Tebet — Recommended (81/100)", got.Headline)

	got, err = ParseGeminiAnswer(raw, &model.Site{}, time.Now())
	require.NoError(t, err)
	assert.Equal(t, `Ruko \u2014 Recommended (81/100)`, got.Headline, "no site name: Gemini's text")
}

func TestGemini_NoKey_NoCall(t *testing.T) {
	c, f := geminiClient(t, fakeAnswer{200, "gemini_ok.json"})
	c.apiKey = ""

	_, _, err := c.GenerateInsight(context.Background(), geminiTestInput())

	assert.ErrorIs(t, err, ErrGeminiNotSetUp)
	assert.Empty(t, f.calls)
}

func TestNewGeminiClient_OnWhenKeySet(t *testing.T) {
	off := NewGeminiClient(nil).(*GeminiClient)
	assert.False(t, off.Configured())
	assert.Equal(t, "gemini-2.5-flash", off.model)
	assert.Equal(t, 60*time.Second, off.http.Timeout)
	assert.Equal(t, GeminiMaxOutputTokens, off.maxOutput)
}

func TestGemini_MaxOutputTokensFromConfig(t *testing.T) {
	c, f := geminiClient(t, fakeAnswer{200, "gemini_ok.json"})
	c.maxOutput = 9000

	_, _, err := c.GenerateInsight(context.Background(), geminiTestInput())

	require.NoError(t, err)
	assert.Equal(t, float64(9000), f.calls[0].body["generationConfig"].(map[string]any)["maxOutputTokens"])
}

func TestGemini_Refusals_NoRetry(t *testing.T) {
	for _, tc := range []struct {
		fixture, code, reason, google string
		status                        int
	}{
		{"gemini_err_key_invalid.json", "API_KEY_INVALID", "kunci Gemini tidak valid", "API key not valid. Please pass a valid API key.", 400},
		{"gemini_err_service_disabled.json", "SERVICE_DISABLED", "Generative Language API belum diizinkan untuk kunci Gemini ini", "Generative Language API has not been used", 403},
		{"gemini_err_quota.json", "RESOURCE_EXHAUSTED", "kuota Gemini di Google untuk kunci ini sudah habis", "You exceeded your current quota", 429},
	} {
		t.Run(tc.code, func(t *testing.T) {
			c, f := geminiClient(t, fakeAnswer{tc.status, tc.fixture})

			_, _, err := c.GenerateInsight(context.Background(), geminiTestInput())

			var denied *GeminiDeniedError
			require.True(t, errors.As(err, &denied), "%v", err)
			assert.Equal(t, tc.status, denied.HTTPStatus)
			assert.Equal(t, tc.code, denied.Code)
			assert.Equal(t, tc.reason, denied.Reason)
			assert.Contains(t, denied.Message, tc.google)
			assert.Len(t, f.calls, 1, "a refusal is not retried")
			assertNoKey(t, err, f)
		})
	}
}

func TestGemini_RetryOnceOn5xxAnd429(t *testing.T) {
	for _, first := range []fakeAnswer{{503, "gemini_err_overloaded.json"}, {429, "gemini_err_rate_limit.json"}} {
		c, f := geminiClient(t, first, fakeAnswer{200, "gemini_ok.json"})
		var usage GeminiUsage

		got, _, err := c.GenerateInsight(WithGeminiUsage(context.Background(), &usage), geminiTestInput())

		require.NoError(t, err, first.fixture)
		assert.Equal(t, 72, got.OverallScore)
		assert.Len(t, f.calls, 2)
		assert.Equal(t, 2, usage.Attempts)
	}
}

func TestGemini_StillFailingAfterRetry(t *testing.T) {
	c, f := geminiClient(t, fakeAnswer{503, "gemini_err_overloaded.json"})

	_, _, err := c.GenerateInsight(context.Background(), geminiTestInput())

	var up *GeminiUpstreamError
	require.True(t, errors.As(err, &up), "%v", err)
	assert.False(t, up.Timeout)
	assert.Equal(t, "gemini unavailable: HTTP 503 UNAVAILABLE The model is overloaded. Please try again later.", err.Error())
	assert.Len(t, f.calls, 2, "one retry at most")
}

func TestGemini_CutAnswer_RetriedOnce(t *testing.T) {
	c, f := geminiClient(t, fakeAnswer{200, "gemini_max_tokens.json"}, fakeAnswer{200, "gemini_ok.json"})
	var usage GeminiUsage

	got, _, err := c.GenerateInsight(WithGeminiUsage(context.Background(), &usage), geminiTestInput())

	require.NoError(t, err)
	assert.Equal(t, 72, got.OverallScore)
	assert.Len(t, f.calls, 2)
	// both tries are billed: tokens added up
	assert.Equal(t, GeminiUsage{Model: "gemini-2.5-flash", PromptTokens: 2 * 3412, OutputTokens: 200 + 1650, ThoughtsTokens: 7800 + 2000, TotalTokens: 11412 + 7062, Attempts: 2}, usage)
}

func TestGemini_CutAndBlockedAnswers(t *testing.T) {
	c, f := geminiClient(t, fakeAnswer{200, "gemini_max_tokens.json"})
	_, _, err := c.GenerateInsight(context.Background(), geminiTestInput())
	var up *GeminiUpstreamError
	require.True(t, errors.As(err, &up), "%v", err)
	assert.Contains(t, err.Error(), "MAX_TOKENS")
	assert.Contains(t, err.Error(), "answer ends: ")
	assert.Len(t, f.calls, 2, "cut twice: one retry at most")

	c, f = geminiClient(t, fakeAnswer{200, "gemini_blocked.json"})
	_, _, err = c.GenerateInsight(context.Background(), geminiTestInput())
	var denied *GeminiDeniedError
	require.True(t, errors.As(err, &denied), "%v", err)
	assert.Equal(t, "PROHIBITED_CONTENT", denied.Code)
	assert.Len(t, f.calls, 1, "a blocked answer is not retried")
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestGemini_NetworkErrors_NoKeyInText(t *testing.T) {
	c, f := geminiClient(t, fakeAnswer{200, "gemini_ok.json"})
	f.netErr = errors.New("dial tcp: connection refused (" + testGeminiKey + ")")

	_, _, err := c.GenerateInsight(context.Background(), geminiTestInput())

	var up *GeminiUpstreamError
	require.True(t, errors.As(err, &up), "%v", err)
	assert.False(t, up.Timeout)
	assert.Len(t, f.calls, 1, "a network error is not retried")
	assert.True(t, strings.Contains(err.Error(), "***"))
	assertNoKey(t, err, f)

	c, f = geminiClient(t, fakeAnswer{200, "gemini_ok.json"})
	f.netErr = timeoutErr{}
	_, _, err = c.GenerateInsight(context.Background(), geminiTestInput())
	require.True(t, errors.As(err, &up), "%v", err)
	assert.True(t, up.Timeout)
}
