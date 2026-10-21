package client

// Prompt, response schema and answer parsing of the Gemini insight, ported
// unchanged from site-dev-api 20eeab4 internal/app/service/gemini_service.go
// (itself a port of the web app's gemini.ts). Only the HTTP call is still a
// TODO (see NewGeminiClient).

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
)

// ifNotEmpty is val, or fallback when val is "" (site-dev-api global.IfNotEmpty).
func ifNotEmpty(val, fallback string) string {
	if val != "" {
		return val
	}
	return fallback
}

// responseSchema mirrors RESPONSE_SCHEMA in gemini.ts field-for-field, so
// Gemini's structured output binds directly into geminiPayload below.
var responseSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"overallScore": map[string]any{"type": "INTEGER", "description": "Skor keseluruhan 0-100 — WAJIB = round(rata-rata 5 metrik)"},
		"recommendation": map[string]any{
			"type": "STRING",
			"enum": []string{"Strongly Recommended", "Recommended", "Proceed With Caution", "Reconsider"},
		},
		"headline":  map[string]any{"type": "STRING", "description": "Format: '{nama site} — {recommendation} ({score}/100)'"},
		"narrative": map[string]any{"type": "STRING", "description": "2-4 kalimat TEGAS. Searah dengan recommendation."},
		"metrics": map[string]any{
			"type": "OBJECT",
			"properties": map[string]any{
				"footTraffic":   metricSchema,
				"demographics":  metricSchema,
				"competition":   metricSchema,
				"accessibility": metricSchema,
				"economic":      metricSchema,
			},
			"required": []string{"footTraffic", "demographics", "competition", "accessibility", "economic"},
		},
		"competitors": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"name":       map[string]any{"type": "STRING"},
					"distanceKm": map[string]any{"type": "NUMBER", "description": "Jarak dalam km, > 0"},
					"type":       map[string]any{"type": "STRING", "description": "Convenience Store / Mini Market / Supermarket"},
				},
				"required": []string{"name", "distanceKm", "type"},
			},
		},
		"risks": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"title":      map[string]any{"type": "STRING"},
					"severity":   map[string]any{"type": "STRING", "enum": []string{"low", "moderate", "high"}},
					"mitigation": map[string]any{"type": "STRING"},
				},
				"required": []string{"title", "severity", "mitigation"},
			},
		},
		"opportunities": map[string]any{
			"type":        "ARRAY",
			"items":       map[string]any{"type": "STRING"},
			"description": "3-5 peluang strategis konkret. Jangan tumpang tindih dengan risks.",
		},
		"confidenceLevel":  map[string]any{"type": "STRING", "enum": []string{"high", "medium", "low"}},
		"confidenceReason": map[string]any{"type": "STRING"},
		"dataAssumptions": map[string]any{
			"type":  "ARRAY",
			"items": map[string]any{"type": "STRING"},
		},
		"contextWarnings": map[string]any{
			"type":        "ARRAY",
			"items":       contextWarningSchema,
			"description": "WAJIB diisi (boleh kosong jika tidak ada masalah).",
		},
	},
	"required": []string{
		"overallScore", "recommendation", "headline", "narrative", "metrics",
		"competitors", "risks", "opportunities", "confidenceLevel",
		"confidenceReason", "dataAssumptions", "contextWarnings",
	},
}

var metricSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"label":   map[string]any{"type": "STRING"},
		"score":   map[string]any{"type": "INTEGER", "description": "Skor 0-100"},
		"band":    map[string]any{"type": "STRING", "enum": []string{"low", "moderate", "good", "high"}},
		"summary": map[string]any{"type": "STRING", "description": "1-2 kalimat ringkas, tegas, tidak mengambang"},
		"factors": map[string]any{
			"type":        "ARRAY",
			"items":       map[string]any{"type": "STRING"},
			"description": "2-4 faktor pendukung konkret, semuanya searah dengan skor",
		},
	},
	"required": []string{"label", "score", "band", "summary", "factors"},
}

var contextWarningSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"field":         map[string]any{"type": "STRING", "description": "Field konteks yang bermasalah"},
		"userClaim":     map[string]any{"type": "STRING"},
		"aiExpectation": map[string]any{"type": "STRING"},
		"severity":      map[string]any{"type": "STRING", "enum": []string{"info", "warning", "critical"}},
		"message":       map[string]any{"type": "STRING"},
	},
	"required": []string{"field", "userClaim", "aiExpectation", "severity", "message"},
}

var parkingLabels = map[string]string{
	constants.ParkingAvailable: "Memadai/luas",
	constants.ParkingLimited:   "Terbatas",
	constants.ParkingNone:      "Tidak ada",
}

// formatContext ports formatContext() in gemini.ts.
func formatContext(siteCtx *model.SiteContext, competitors []*model.SiteCompetitor, notes []*model.SiteCustomNote) string {
	if siteCtx == nil {
		return "(Tidak ada observasi on-site dari tim Site Development.)"
	}

	var lines []string
	if siteCtx.FootTrafficObservation != "" {
		lines = append(lines, fmt.Sprintf("- Foot traffic teramati: %s", siteCtx.FootTrafficObservation))
	}
	if siteCtx.PeakHours != "" {
		lines = append(lines, fmt.Sprintf("- Jam sibuk: %s", siteCtx.PeakHours))
	}
	if siteCtx.ParkingAvailability != nil && *siteCtx.ParkingAvailability != "" {
		label := parkingLabels[*siteCtx.ParkingAvailability]
		if label == "" {
			label = *siteCtx.ParkingAvailability
		}
		lines = append(lines, fmt.Sprintf("- Ketersediaan parkir: %s", label))
	}
	if siteCtx.PublicTransport != "" {
		lines = append(lines, fmt.Sprintf("- Transportasi umum: %s", siteCtx.PublicTransport))
	}
	if siteCtx.AnchorLandmark != "" {
		lines = append(lines, fmt.Sprintf("- Anchor/landmark sekitar: %s", siteCtx.AnchorLandmark))
	}
	if siteCtx.ObservedDemographics != "" {
		lines = append(lines, fmt.Sprintf("- Demografi teramati: %s", siteCtx.ObservedDemographics))
	}
	if len(competitors) > 0 {
		lines = append(lines, "- Kompetitor TERAMATI di lapangan (data ini WAJIB digunakan persis, jangan ganti):")
		for _, c := range competitors {
			name := c.CompetitorName
			if name == "" {
				name = "(tanpa nama)"
			}
			lines = append(lines, fmt.Sprintf("    • %s — %s — %s", name, c.Distance, c.CompetitorType))
		}
	}
	if len(notes) > 0 {
		lines = append(lines, "- Catatan tambahan dari tim lapangan:")
		for _, n := range notes {
			title := n.Title
			if title == "" {
				title = "(tanpa label)"
			}
			content := n.Content
			if content == "" {
				content = "(kosong)"
			}
			lines = append(lines, fmt.Sprintf("    • %s: %s", title, content))
		}
	}

	if len(lines) == 0 {
		return "(Tim Site Development belum mengisi observasi detail.)"
	}
	return strings.Join(lines, "\n")
}

func hasContext(siteCtx *model.SiteContext, competitors []*model.SiteCompetitor, notes []*model.SiteCustomNote) bool {
	if siteCtx == nil {
		return false
	}
	return siteCtx.FootTrafficObservation != "" ||
		siteCtx.PeakHours != "" ||
		(siteCtx.ParkingAvailability != nil && *siteCtx.ParkingAvailability != "") ||
		siteCtx.PublicTransport != "" ||
		siteCtx.AnchorLandmark != "" ||
		siteCtx.ObservedDemographics != "" ||
		len(competitors) > 0 ||
		len(notes) > 0
}

func yesNo(b bool) string {
	if b {
		return "Ya"
	}
	return "Tidak"
}

// buildPrompt ports buildPrompt() in gemini.ts.
func buildPrompt(site *model.Site, siteCtx *model.SiteContext, competitors []*model.SiteCompetitor, notes []*model.SiteCustomNote) string {
	addressParts := make([]string, 0, 3)
	for _, p := range []string{site.SiteAddress1, site.SiteAddress2, site.SiteAddress3} {
		if p != "" {
			addressParts = append(addressParts, p)
		}
	}
	address := strings.Join(addressParts, ", ")
	if address == "" {
		address = "—"
	}
	contextBlock := formatContext(siteCtx, competitors, notes)
	hasCtx := hasContext(siteCtx, competitors, notes)

	region := ifNotEmpty(site.SiteRegion, "—")
	postalCode := ifNotEmpty(site.SitePostalCode, "—")
	district := ifNotEmpty(site.SiteDistrict, "—")
	salesArea := "tidak disebutkan"
	if site.SiteSalesArea > 0 {
		salesArea = fmt.Sprintf("%.0f m²", site.SiteSalesArea)
	}
	plannedOpening := "—"
	if !site.SitePlannedOpeningDate.IsZero() {
		plannedOpening = site.SitePlannedOpeningDate.Format("2006-01-02")
	}

	name := site.SiteName1
	if site.SiteName2 != "" {
		name = fmt.Sprintf("%s (%s)", site.SiteName1, site.SiteName2)
	}

	ctxRule := "Karena TIDAK ada konteks user, generate estimasi umum dari kota/koordinat. contextWarnings boleh kosong, tapi confidenceLevel WAJIB 'low' atau 'medium' dengan reason yang jelas."
	if hasCtx {
		ctxRule = "Karena ADA konteks user, validasi setiap klaim dan isi contextWarnings sesuai aturan A."
	}

	return fmt.Sprintf(`Anda adalah analis senior site selection FamilyMart Indonesia. Tugas: hasilkan analisis kelayakan lokasi yang TEGAS, KONSISTEN, dan TIDAK KONTRADIKTIF.

============================
DATA LOKASI (otoritatif dari sistem):
============================
- Nama site: %s
- Alamat: %s
- Kota: %s
- Region: %s
- Kode pos: %s
- District: %s
- Koordinat: %v, %v
- Sales area: %s
- Buka 24 jam: %s
- Menjual alkohol: %s
- Rencana buka: %s

============================
KLAIM OBSERVASI ON-SITE OLEH TIM SITE DEVELOPMENT (USER CONTEXT):
============================
%s

============================
ATURAN ANALISIS (WAJIB DIPATUHI):
============================

A) GROUND TRUTH & VALIDASI KLAIM USER
   1. Klaim user di "USER CONTEXT" diprioritaskan SEBAGAI INPUT — namun WAJIB divalidasi terhadap realita lokasi (kota/koordinat/karakteristik area).
   2. Jika klaim user kontradiktif dengan apa yang umumnya benar di lokasi tersebut, WAJIB tambahkan entri ke "contextWarnings" — JANGAN diam.
   3. Klaim janggal/ngawur yang harus selalu di-flag: kompetitor jarak 0 km atau > 5 km untuk kategori convenience; foot traffic tanpa angka kuantitatif; "tidak ada kompetitor" di area perkotaan padat; "foot traffic tinggi" tapi area residensial sepi; parkir "memadai" tapi alamat gang sempit; anchor/landmark tidak match kota; demografi bertentangan dengan profil kota.
   4. severity: "critical" = klaim secara fisik tidak mungkin/pasti salah; "warning" = kontradiktif dengan realita umum, perlu re-survey; "info" = kurang spesifik/format tidak ideal.
   5. Walaupun ada warning, TETAP gunakan klaim user untuk menghitung metrik — flag adalah peringatan untuk reviewer, bukan auto-reject.
   6. Jika tidak ada masalah, kembalikan "contextWarnings": [].

B) INTERNAL CONSISTENCY (WAJIB)
   1. overallScore = round(average dari 5 metrik). Tidak boleh menyimpang lebih dari 1 poin.
   2. recommendation HARUS sesuai band overallScore: ≥82 "Strongly Recommended"; 70-81 "Recommended"; 55-69 "Proceed With Caution"; <55 "Reconsider".
   3. band setiap metrik HARUS sesuai score: ≥85 "high", 70-84 "good", 50-69 "moderate", <50 "low".
   4. narrative HARUS searah dengan recommendation. JANGAN campur "sangat menjanjikan" dengan "tidak layak" di paragraf yang sama.
   5. Setiap metric.factors HARUS searah dengan score-nya.
   6. risks dan opportunities tidak boleh saling membatalkan.
   7. Severity risk harus match dampak.

C) GAYA & DETAIL
   1. Bahasa Indonesia profesional retail. Tegas, tidak bertele-tele.
   2. Setiap angka di summary HARUS punya basis (mis. estimasi berdasarkan koridor utama %s).
   3. Headline FIX: "%s — {recommendation} ({score}/100)".
   4. confidenceLevel: "high" = konteks lengkap & konsisten; "medium" = parsial/minor warning; "low" = tanpa konteks atau ada warning severity critical/warning.
   5. dataAssumptions: 3-5 asumsi konkret yang Anda pakai.

D) %s

============================
OUTPUT:
============================
Kembalikan HANYA JSON sesuai schema. Jangan markdown fence, jangan komentar.`,
		name, address, site.SiteCity, region, postalCode, district,
		site.SiteLat, site.SiteLon, salesArea, yesNo(site.SiteOpen24Hours), yesNo(site.SiteSellingAlcohol), plannedOpening,
		contextBlock, site.SiteCity, site.SiteName1, ctxRule,
	)
}

type geminiRequestBody struct {
	Contents         []geminiContent `json:"contents"`
	GenerationConfig struct {
		ResponseMimeType string  `json:"responseMimeType"`
		ResponseSchema   any     `json:"responseSchema"`
		Temperature      float64 `json:"temperature"`
		MaxOutputTokens  int     `json:"maxOutputTokens"`
	} `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponseBody struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// geminiPayload is the raw JSON shape Gemini returns (RESPONSE_SCHEMA), sans
// generatedAt/id/site_id — those are attached by the caller.
type geminiPayload struct {
	OverallScore     int                           `json:"overallScore"`
	Recommendation   string                        `json:"recommendation"`
	Headline         string                        `json:"headline"`
	Narrative        string                        `json:"narrative"`
	Metrics          dto.AIInsightMetrics          `json:"metrics"`
	Competitors      []dto.AIInsightCompetitor     `json:"competitors"`
	Risks            []dto.AIInsightRisk           `json:"risks"`
	Opportunities    []string                      `json:"opportunities"`
	ConfidenceLevel  string                        `json:"confidenceLevel"`
	ConfidenceReason string                        `json:"confidenceReason"`
	DataAssumptions  []string                      `json:"dataAssumptions"`
	ContextWarnings  []dto.AIInsightContextWarning `json:"contextWarnings"`
}

// GeminiRequest is the JSON body site-dev-api posts to generateContent for
// in (prompt + response schema, temperature 0.35, max 8000 tokens).
func GeminiRequest(in GeminiInput) ([]byte, error) {
	body := geminiRequestBody{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: buildPrompt(in.Site, in.Context, in.Competitors, in.Notes)}}}},
	}
	body.GenerationConfig.ResponseMimeType = "application/json"
	body.GenerationConfig.ResponseSchema = responseSchema
	body.GenerationConfig.Temperature = 0.35
	body.GenerationConfig.MaxOutputTokens = 8000

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal request: %w", err)
	}
	return payload, nil
}

// ParseGeminiAnswer reads a 200 answer of generateContent into the insight
// of site (GeneratedAt = now), same as site-dev-api.
func ParseGeminiAnswer(respBytes []byte, site *model.Site, now time.Time) (dto.AIInsightResponseGet, error) {
	var parsed geminiResponseBody
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return dto.AIInsightResponseGet{}, fmt.Errorf("gemini: unmarshal response envelope: %w", err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 || parsed.Candidates[0].Content.Parts[0].Text == "" {
		return dto.AIInsightResponseGet{}, fmt.Errorf("gemini returned empty response")
	}

	var out geminiPayload
	if err := json.Unmarshal([]byte(parsed.Candidates[0].Content.Parts[0].Text), &out); err != nil {
		return dto.AIInsightResponseGet{}, fmt.Errorf("gemini: unmarshal insight JSON: %w", err)
	}

	return dto.AIInsightResponseGet{
		SiteID:           site.SiteID,
		OverallScore:     out.OverallScore,
		Recommendation:   out.Recommendation,
		Headline:         out.Headline,
		Narrative:        out.Narrative,
		Metrics:          out.Metrics,
		Competitors:      out.Competitors,
		Risks:            out.Risks,
		Opportunities:    out.Opportunities,
		ConfidenceLevel:  out.ConfidenceLevel,
		ConfidenceReason: out.ConfidenceReason,
		DataAssumptions:  out.DataAssumptions,
		ContextWarnings:  out.ContextWarnings,
		GeneratedAt:      now,
	}, nil
}
