package client

import (
	"encoding/json"
	"os"
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

	p := buildPrompt(site, nil, nil, nil, nil)

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
	assert.Equal(t, float64(12000), cfg["maxOutputTokens"]) // site-dev-api 8000: cut answers on 2.5-flash (thinking counts)

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

func promptFixtureSite() (*model.Site, *model.SiteContext, []*model.SiteCompetitor, []*model.SiteCustomNote) {
	park := "LIMITED"
	site := &model.Site{SiteID: "ST20261016000001", SiteName1: "Kemang Timur", SiteName2: "Ruko", SiteAddress1: "Jl. Kemang Timur No. 5", SiteCity: "Jakarta Selatan", SiteRegion: "DKI Jakarta", SitePostalCode: "12730", SiteDistrict: "Mampang Prapatan", SiteLat: -6.262, SiteLon: 106.8167, SiteSalesArea: 120, SiteOpen24Hours: true, SitePlannedOpeningDate: time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)}
	ctx := &model.SiteContext{FootTrafficObservation: "±300 orang/jam sore", ParkingAvailability: &park, AnchorLandmark: "Kemang Village"}
	comps := []*model.SiteCompetitor{{CompetitorName: "Indomaret", Distance: "150 m", CompetitorType: "Convenience Store"}}
	notes := []*model.SiteCustomNote{{Title: "Sewa", Content: "Rp 250 jt/tahun"}}
	return site, ctx, comps, notes
}

// Without map data the prompt is byte for byte site-dev-api 20eeab4's
// (golden files captured from that code in site-dev-api's local commit
// 2191e07, copied unchanged).
func TestPromptSameAsSiteDevAPI(t *testing.T) {
	site, ctx, comps, notes := promptFixtureSite()
	want, err := os.ReadFile("testdata/prompt_original_with_context.golden")
	require.NoError(t, err)
	assert.Equal(t, novemberPrompt(string(want)), buildPrompt(site, ctx, comps, notes, nil))
	want, err = os.ReadFile("testdata/prompt_original_no_context.golden")
	require.NoError(t, err)
	assert.Equal(t, novemberPrompt(string(want)), buildPrompt(site, nil, nil, nil, nil))
}

// novemberPrompt is the 20eeab4 prompt with exactly the November changes
// (API-CHANGES §13.5): the B1 score formula with the 6th metric and rule H
// after rule D. Nothing else may differ.
func novemberPrompt(old string) string {
	s := strings.Replace(old, "   1. overallScore = round(average dari 5 metrik). Tidak boleh menyimpang lebih dari 1 poin.",
		"   1. overallScore = round(0.18 × (footTraffic + demographics + competition + accessibility + economic) + 0.10 × vehicleTraffic). Tidak boleh menyimpang lebih dari 1 poin. (Lalu lintas kendaraan bobotnya lebih kecil: faktor utama, tapi tidak boleh mendominasi.)", 1)
	return strings.Replace(s, "\n\n============================\nOUTPUT:", "\n\n"+ruleH+"\n\n============================\nOUTPUT:", 1)
}

func TestPrompt_NovemberSurveyFields(t *testing.T) {
	site, ctx, comps, notes := promptFixtureSite()
	busy, pnote, cnote, tnote := "BUSY", "parkir motor 6", "Alfamart tutup", "bus sering berhenti"
	n, m := 120, 10
	ctx.VehicleTrafficLevel, ctx.VehicleTrafficCount, ctx.VehicleTrafficMinutes, ctx.VehicleTrafficNote = &busy, &n, &m, &tnote
	ctx.ParkingNote, ctx.CompetitorNote = &pnote, &cnote

	p := buildPrompt(site, ctx, comps, notes, nil)

	for _, want := range []string{
		"- Lalu lintas kendaraan di jalan depan lokasi: Ramai; dihitung 120 kendaraan dalam 10 menit (± 720 kendaraan/jam)",
		"    • Catatan lalu lintas: bus sering berhenti",
		"- Ketersediaan parkir: Terbatas\n    • Catatan parkir: parkir motor 6",
		`- CATATAN KOMPETITOR DARI SITE STAFF (teks bebas, BELUM DIVERIFIKASI, lihat aturan G): "Alfamart tutup"`,
		"G) CATATAN KOMPETITOR STAF:",
		"H) LALU LINTAS KENDARAAN",
	} {
		assert.Contains(t, p, want)
	}
	assert.Contains(t, string(mustJSON(t, responseSchema)), `"vehicleTraffic"`)
	// only the traffic level counts as context
	assert.True(t, hasContext(&model.SiteContext{VehicleTrafficLevel: &busy}, nil, nil))
}

func mustJSON(t *testing.T, v any) []byte {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
