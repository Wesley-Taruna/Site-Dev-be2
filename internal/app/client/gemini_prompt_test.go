package client

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatContext(t *testing.T) {
	assert.Equal(t, "(Tidak ada observasi on-site dari tim Site Development.)", formatContext(nil, nil, nil))
	assert.Equal(t, "(Tim Site Development belum mengisi observasi detail.)", formatContext(&model.SiteContext{}, nil, nil))

	limited := "LIMITED"
	got := formatContext(&model.SiteContext{PeakHours: "17-20", ParkingAvailability: &limited},
		[]*model.SiteCompetitor{{CompetitorName: "", Distance: "50m", CompetitorType: "minimarket"}},
		[]*model.SiteCustomNote{{Title: "Banjir"}})
	assert.Equal(t, "- Jam sibuk: 17-20\n- Ketersediaan parkir: Terbatas\n- Kompetitor TERAMATI di lapangan (data ini WAJIB digunakan persis, jangan ganti):\n    • (tanpa nama) — 50m — minimarket\n- Catatan tambahan dari tim lapangan:\n    • Banjir: (kosong)", got)
}

func TestBuildPrompt(t *testing.T) {
	site := &model.Site{SiteName1: "Toko Kemang", SiteName2: "Kemang 2", SiteAddress1: "Jl. Kemang", SiteCity: "Jakarta", SiteLat: -6.26, SiteLon: 106.81,
		SiteSalesArea: 120, SiteOpen24Hours: true, SitePlannedOpeningDate: time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)}

	p := buildPrompt(site, nil, nil, nil)

	assert.True(t, strings.HasPrefix(p, "Anda adalah analis senior site selection FamilyMart Indonesia."))
	for _, want := range []string{
		"- Nama site: Toko Kemang (Kemang 2)", "- Alamat: Jl. Kemang", "- Region: —", "- Koordinat: -6.26, 106.81",
		"- Sales area: 120 m²", "- Buka 24 jam: Ya", "- Menjual alkohol: Tidak", "- Rencana buka: 2027-01-15",
		`Headline FIX: "Toko Kemang — {recommendation} ({score}/100)".`,
		"D) Karena TIDAK ada konteks user",
	} {
		assert.Contains(t, p, want)
	}
}

func TestGeminiRequestAndAnswer(t *testing.T) {
	body, err := GeminiRequest(GeminiInput{Site: &model.Site{SiteName1: "X"}})
	require.NoError(t, err)
	var req map[string]any
	require.NoError(t, json.Unmarshal(body, &req))
	cfg := req["generationConfig"].(map[string]any)
	assert.Equal(t, "application/json", cfg["responseMimeType"])
	assert.Equal(t, 0.35, cfg["temperature"])
	assert.Equal(t, float64(8000), cfg["maxOutputTokens"])

	answer := `{"candidates":[{"content":{"parts":[{"text":"{\"overallScore\":74,\"recommendation\":\"Recommended\",\"metrics\":{\"footTraffic\":{\"score\":80,\"factors\":[\"a\"]}},\"opportunities\":[\"x\"]}"}]}}]}`
	now := time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)
	insight, err := ParseGeminiAnswer([]byte(answer), &model.Site{SiteID: "ST1"}, now)
	require.NoError(t, err)
	assert.Equal(t, "ST1", insight.SiteID)
	assert.Equal(t, 74, insight.OverallScore)
	assert.Equal(t, 80, insight.Metrics.FootTraffic.Score)
	assert.Equal(t, now, insight.GeneratedAt)

	_, err = ParseGeminiAnswer([]byte(`{"candidates":[]}`), &model.Site{}, now)
	assert.EqualError(t, err, "gemini returned empty response")
}
