package client

import (
	"context"
	"errors"

	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/env"
)

// ErrGeminiNotSetUp is returned until the real Gemini client exists.
var ErrGeminiNotSetUp = errors.New("gemini: client is not set up yet (TODO, plan Q4)")

// GeminiInput is what the insight is built from: the site, its on-site
// survey (nil when never filled in) and the survey lists.
type GeminiInput struct {
	Site        *model.Site
	Context     *model.SiteContext
	Competitors []*model.SiteCompetitor
	Notes       []*model.SiteCustomNote
}

// IGeminiClient turns a site into a feasibility insight with Google Gemini.
// Only the backend calls Gemini; the key never leaves the server.
type IGeminiClient interface {
	// GenerateInsight returns the insight (SiteID, GeneratedAt filled) and
	// the Gemini model name used.
	GenerateInsight(ctx context.Context, in GeminiInput) (insight dto.AIInsightResponseGet, geminiModel string, err error)
}

// NewGeminiClient returns the Gemini client.
//
// TODO(plan Q4, needs Wes's OK): the real call. site-dev-api posts
// GeminiRequest(in) (prompt and response schema in gemini_prompt.go, ported
// unchanged) to generativelanguage.googleapis.com
// /v1beta/models/<gemini.model, default gemini-2.5-flash>:generateContent
// with gemini.apikey, temperature 0.35, max 8000 tokens, 60 s timeout, then
// reads candidates[0].content.parts[0].text as the insight JSON
// (ParseGeminiAnswer). When built: send the key in the x-goog-api-key
// header, not in the URL, so it cannot leak into error messages (our local
// fix e4d8c60). Until then every call answers ErrGeminiNotSetUp.
func NewGeminiClient(_ env.Provider) IGeminiClient {
	return pendingGeminiClient{}
}

type pendingGeminiClient struct{}

func (pendingGeminiClient) GenerateInsight(context.Context, GeminiInput) (dto.AIInsightResponseGet, string, error) {
	return dto.AIInsightResponseGet{}, "", ErrGeminiNotSetUp
}
