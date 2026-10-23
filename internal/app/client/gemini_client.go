package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/env"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// ErrGeminiNotSetUp: no Gemini key in the config (gemini.apikey); the AI is
// off and nothing is sent to Google.
var ErrGeminiNotSetUp = errors.New("gemini: key is not set (gemini.apikey)")

// GeminiDeniedError: Google refused the call for a reason a retry does not
// fix (key invalid, API not enabled, quota used up, model unknown, answer
// blocked). Reason is plain Indonesian for the answer; Code is Google's
// status/reason word; Message is Google's own text (key removed).
type GeminiDeniedError struct {
	HTTPStatus int
	Code       string
	Reason     string
	Message    string
}

func (e *GeminiDeniedError) Error() string {
	s := fmt.Sprintf("gemini denied (HTTP %d %s): %s", e.HTTPStatus, e.Code, e.Reason)
	if e.Message != "" {
		s += "; Google: " + e.Message
	}
	return s
}

// GeminiUpstreamError: Gemini could not give an answer now (5xx / 429 after
// the retry, network error, timeout, unreadable or cut answer). Timeout is
// true when Gemini did not answer in time.
type GeminiUpstreamError struct {
	Timeout bool
	Detail  string
}

func (e *GeminiUpstreamError) Error() string { return "gemini unavailable: " + e.Detail }

// GeminiInput is what the insight is built from: the site, its on-site
// survey (nil when never filled in) and the survey lists.
type GeminiInput struct {
	Site        *model.Site
	Context     *model.SiteContext
	Competitors []*model.SiteCompetitor
	Notes       []*model.SiteCustomNote
	// Measured is the nearby map data (local §6); nil = the original prompt.
	Measured *dto.NearbyMeasured
}

// GeminiUsage is the token count of one answer (Google's usageMetadata), for
// the cost log. Thoughts are billed as output.
type GeminiUsage struct {
	Model          string
	PromptTokens   int
	OutputTokens   int
	ThoughtsTokens int
	TotalTokens    int
	Attempts       int
}

type geminiUsageKey struct{}

// WithGeminiUsage returns a ctx in which GenerateInsight writes the token
// count of its answer into u (keeps IGeminiClient unchanged).
func WithGeminiUsage(ctx context.Context, u *GeminiUsage) context.Context {
	return context.WithValue(ctx, geminiUsageKey{}, u)
}

// IGeminiClient turns a site into a feasibility insight with Google Gemini.
// Only the backend calls Gemini; the key never leaves the server.
type IGeminiClient interface {
	// GenerateInsight returns the insight (SiteID, GeneratedAt filled) and
	// the Gemini model name used.
	GenerateInsight(ctx context.Context, in GeminiInput) (insight dto.AIInsightResponseGet, geminiModel string, err error)
}

const (
	geminiBaseURL      = "https://generativelanguage.googleapis.com/v1beta/models"
	geminiDefaultModel = "gemini-2.5-flash"
	// geminiMaxBody: an insight answer is ~10-20 KB.
	geminiMaxBody = 5 << 20
)

var geminiTracer = otel.Tracer("site-dev-backend/client")

// GeminiClient is Google AI Studio's Gemini API (generateContent), standard
// library only, same call as site-dev-api's gemini_service.go. The key goes
// in the x-goog-api-key header, never in a URL, a log or an error (local
// fix e4d8c60).
type GeminiClient struct {
	apiKey    string
	model     string
	baseURL   string
	http      *http.Client
	maxOutput int
	retryWait time.Duration
	now       func() time.Time
}

// NewGeminiClient reads gemini.apikey (devops' secret store; empty = AI off,
// every call answers ErrGeminiNotSetUp), gemini.model (gemini-2.5-flash),
// gemini.timeout_seconds (60, per try) and gemini.max_output_tokens
// (GeminiMaxOutputTokens; thinking + answer).
func NewGeminiClient(env env.Provider) IGeminiClient {
	c := &GeminiClient{
		model:     geminiDefaultModel,
		baseURL:   geminiBaseURL,
		http:      &http.Client{Timeout: 60 * time.Second},
		maxOutput: GeminiMaxOutputTokens,
		retryWait: 2 * time.Second,
		now:       time.Now,
	}
	if env != nil {
		c.apiKey = strings.TrimSpace(env.GetString("gemini.apikey"))
		c.model = ifNotEmpty(strings.TrimSpace(env.GetString("gemini.model")), geminiDefaultModel)
		if n := env.GetInt("gemini.timeout_seconds"); n > 0 {
			c.http.Timeout = time.Duration(n) * time.Second
		}
		if n := env.GetInt("gemini.max_output_tokens"); n > 0 {
			c.maxOutput = n
		}
	}
	return c
}

// Configured: a key is set (gemini.apikey).
func (c *GeminiClient) Configured() bool { return c.apiKey != "" }

// GenerateInsight posts GeminiRequest(in) (prompt + response schema,
// temperature 0.35, gemini.max_output_tokens) to <model>:generateContent and reads
// candidates[0].content.parts[0].text with ParseGeminiAnswer. One retry at
// most, only on HTTP 5xx, a 429 rate limit (not a used-up quota) or an
// answer cut off at the token budget (MAX_TOKENS).
func (c *GeminiClient) GenerateInsight(ctx context.Context, in GeminiInput) (insight dto.AIInsightResponseGet, geminiModel string, err error) {
	ctx, span := geminiTracer.Start(ctx, "GeminiClient.GenerateInsight")
	defer span.End()
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
	}()
	span.SetAttributes(attribute.String("gemini.model", c.model))

	if !c.Configured() {
		return insight, "", ErrGeminiNotSetUp
	}
	if in.Site == nil {
		return insight, "", errors.New("gemini: no site")
	}
	maxOutput := c.maxOutput
	if maxOutput <= 0 {
		maxOutput = GeminiMaxOutputTokens
	}
	payload, err := geminiRequest(in, maxOutput)
	if err != nil {
		return insight, "", err
	}

	// One retry at most: on HTTP 5xx / 429 rate limit, or on an answer cut
	// off at the token budget (live check 23 Oct: gemini-2.5-flash
	// sometimes loops on "\n" inside a JSON string until the budget ends).
	usage := GeminiUsage{Model: c.model}
	defer func() {
		span.SetAttributes(
			attribute.Int("gemini.attempts", usage.Attempts),
			attribute.Int("gemini.tokens.prompt", usage.PromptTokens),
			attribute.Int("gemini.tokens.output", usage.OutputTokens),
			attribute.Int("gemini.tokens.thoughts", usage.ThoughtsTokens),
		)
		if u, ok := ctx.Value(geminiUsageKey{}).(*GeminiUsage); ok && u != nil && usage.TotalTokens > 0 {
			*u = usage
		}
	}()
	for {
		usage.Attempts++
		raw, retry, callErr := c.call(ctx, payload)
		err = callErr
		if err == nil {
			var meta geminiAnswerMeta
			_ = json.Unmarshal(raw, &meta)
			// billed per try: add up
			usage.PromptTokens += meta.UsageMetadata.PromptTokenCount
			usage.OutputTokens += meta.UsageMetadata.CandidatesTokenCount
			usage.ThoughtsTokens += meta.UsageMetadata.ThoughtsTokenCount
			usage.TotalTokens += meta.UsageMetadata.TotalTokenCount
			if err = answerProblem(meta); err == nil {
				insight, err = ParseGeminiAnswer(raw, in.Site, c.now())
				if err != nil {
					return dto.AIInsightResponseGet{}, "", &GeminiUpstreamError{Detail: err.Error()}
				}
				return insight, c.model, nil
			}
			var up *GeminiUpstreamError
			retry = errors.As(err, &up) && strings.Contains(up.Detail, "MAX_TOKENS")
			if retry {
				up.Detail += "; answer ends: " + answerTail(raw, 120)
			}
		}
		if !retry || usage.Attempts >= 2 {
			return insight, "", err
		}
		select {
		case <-ctx.Done():
			return insight, "", &GeminiUpstreamError{Timeout: true, Detail: "stopped: " + ctx.Err().Error()}
		case <-time.After(c.retryWait):
		}
	}
}

// geminiAnswerMeta is the part of a 200 answer besides the text: why it
// stopped, a blocked prompt, the token count.
type geminiAnswerMeta struct {
	Candidates []struct {
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		ThoughtsTokenCount   int `json:"thoughtsTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// answerProblem: a 200 answer that holds no usable insight.
func answerProblem(m geminiAnswerMeta) error {
	if m.PromptFeedback.BlockReason != "" {
		return &GeminiDeniedError{HTTPStatus: http.StatusOK, Code: m.PromptFeedback.BlockReason,
			Reason: "Gemini menolak menjawab permintaan ini (" + m.PromptFeedback.BlockReason + ")"}
	}
	if len(m.Candidates) == 0 {
		return nil // ParseGeminiAnswer says "empty response"
	}
	switch fr := m.Candidates[0].FinishReason; fr {
	case "", "STOP":
		return nil
	case "MAX_TOKENS":
		return &GeminiUpstreamError{Detail: "answer cut off (MAX_TOKENS)"}
	default: // SAFETY, RECITATION, BLOCKLIST, PROHIBITED_CONTENT, OTHER, …
		return &GeminiDeniedError{HTTPStatus: http.StatusOK, Code: fr,
			Reason: "Gemini menghentikan jawaban (" + fr + ")"}
	}
}

type googleGeminiError struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
		Details []struct {
			Reason string `json:"reason"`
		} `json:"details"`
	} `json:"error"`
}

// call sends one generateContent request. retry = the failure may pass on a
// second try (5xx, or a 429 rate limit that is not a used-up quota).
func (c *GeminiClient) call(ctx context.Context, payload []byte) (raw []byte, retry bool, err error) {
	ctx, span := geminiTracer.Start(ctx, "GeminiClient.generateContent")
	defer span.End()
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
	}()

	endpoint := c.baseURL + "/" + url.PathEscape(c.model) + ":generateContent"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, false, fmt.Errorf("gemini: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		// keep only the cause (same rule as the other clients)
		var uErr *url.Error
		if errors.As(err, &uErr) {
			err = uErr.Err
		}
		timeout := errors.Is(err, context.DeadlineExceeded)
		var nErr interface{ Timeout() bool }
		if errors.As(err, &nErr) && nErr.Timeout() {
			timeout = true
		}
		detail := "request failed: " + c.redact(err.Error())
		if timeout {
			detail = fmt.Sprintf("no answer within %s", c.http.Timeout)
		}
		return nil, false, &GeminiUpstreamError{Timeout: timeout, Detail: detail}
	}
	defer resp.Body.Close()
	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	raw, err = io.ReadAll(io.LimitReader(resp.Body, geminiMaxBody))
	if err != nil {
		return nil, false, &GeminiUpstreamError{Detail: "read answer: " + c.redact(err.Error())}
	}
	if resp.StatusCode == http.StatusOK {
		return raw, false, nil
	}

	var ge googleGeminiError
	_ = json.Unmarshal(raw, &ge)
	code := ""
	for _, d := range ge.Error.Details {
		if d.Reason != "" {
			code = d.Reason
			break
		}
	}
	if code == "" {
		code = ge.Error.Status
	}
	msg := c.redact(ge.Error.Message)
	if len(msg) > 300 {
		msg = msg[:300]
	}

	switch {
	case resp.StatusCode == http.StatusTooManyRequests && quotaUsedUp(msg):
		return nil, false, &GeminiDeniedError{HTTPStatus: resp.StatusCode, Code: code, Message: msg,
			Reason: "kuota Gemini di Google untuk kunci ini sudah habis"}
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return nil, true, &GeminiUpstreamError{Detail: strings.TrimSpace(fmt.Sprintf("HTTP %d %s %s", resp.StatusCode, code, msg))}
	}
	return nil, false, &GeminiDeniedError{HTTPStatus: resp.StatusCode, Code: code, Message: msg,
		Reason: geminiDeniedReason(resp.StatusCode, code)}
}

// quotaUsedUp: a 429 that a retry in 2 s cannot fix (daily / free-tier
// quota, billing) rather than a short rate limit.
func quotaUsedUp(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "quota") && (strings.Contains(m, "exceeded") || strings.Contains(m, "billing"))
}

// geminiDeniedReason is the Indonesian text for a refusal.
func geminiDeniedReason(status int, code string) string {
	switch code {
	case "API_KEY_INVALID":
		return "kunci Gemini tidak valid"
	case "SERVICE_DISABLED", "API_KEY_SERVICE_BLOCKED":
		return "Generative Language API belum diizinkan untuk kunci Gemini ini"
	case "API_KEY_HTTP_REFERRER_BLOCKED", "API_KEY_IP_ADDRESS_BLOCKED":
		return "kunci Gemini dibatasi dan tidak boleh dipakai dari server ini"
	}
	switch status {
	case http.StatusNotFound:
		return "model Gemini tidak dikenal (gemini.model)"
	case http.StatusUnauthorized, http.StatusForbidden:
		return "Google menolak kunci Gemini (" + ifNotEmpty(code, fmt.Sprint(status)) + ")"
	}
	return strings.TrimSpace(fmt.Sprintf("Google menolak permintaan Gemini (HTTP %d %s)", status, code))
}

// redact removes the key from any text that could reach a log or an error.
func (c *GeminiClient) redact(s string) string {
	if c.apiKey == "" {
		return s
	}
	return strings.ReplaceAll(s, c.apiKey, "***")
}

// answerTail is the end of the answer text (site data only, no secrets), to
// see where a cut answer ran away.
func answerTail(raw []byte, n int) string {
	var body geminiResponseBody
	if json.Unmarshal(raw, &body) != nil || len(body.Candidates) == 0 || len(body.Candidates[0].Content.Parts) == 0 {
		return "(none)"
	}
	r := []rune(body.Candidates[0].Content.Parts[0].Text)
	if len(r) > n {
		r = r[len(r)-n:]
	}
	return strconv.Quote(string(r))
}
