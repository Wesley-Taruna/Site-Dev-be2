package client

// Prompt, response schema and answer parsing of the Gemini insight, ported
// unchanged from site-dev-api 20eeab4 internal/app/service/gemini_service.go
// (itself a port of the web app's gemini.ts). The HTTP call is in
// gemini_client.go.

import (
	"encoding/json"
	"fmt"
	"math"
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
		"overallScore": map[string]any{"type": "INTEGER", "description": "Skor keseluruhan 0-100 — WAJIB = " + overallScoreFormula},
		"recommendation": map[string]any{
			"type": "STRING",
			"enum": []string{"Strongly Recommended", "Recommended", "Proceed With Caution", "Reconsider"},
		},
		"headline":  map[string]any{"type": "STRING", "description": "Format: '{nama site} — {recommendation} ({score}/100)'"},
		"narrative": map[string]any{"type": "STRING", "description": "2-4 kalimat TEGAS. Searah dengan recommendation."},
		"metrics": map[string]any{
			"type": "OBJECT",
			"properties": map[string]any{
				"footTraffic":    metricSchema,
				"vehicleTraffic": metricSchema,
				"demographics":   metricSchema,
				"competition":    metricSchema,
				"accessibility":  metricSchema,
				"economic":       metricSchema,
			},
			"required": []string{"footTraffic", "vehicleTraffic", "demographics", "competition", "accessibility", "economic"},
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

// overallScoreFormula is the overall score with the 6th metric (November
// scope, API-CHANGES §13.5; weights in constants, open for Wes). Was
// "round(rata-rata 5 metrik)".
var overallScoreFormula = "round(" + constants.ScoreWeightMainMetric + " × (footTraffic + demographics + competition + accessibility + economic) + " +
	constants.ScoreWeightVehicleTraffic + " × vehicleTraffic)"

// trafficLabels are the prompt words of the vehicle traffic levels
// (prototype src/lib/traffic.ts TRAFFIC_LEVELS .ai).
var trafficLabels = map[string]string{
	constants.VehicleTrafficQuiet:    "Sepi (jarang kendaraan lewat)",
	constants.VehicleTrafficModerate: "Sedang",
	constants.VehicleTrafficBusy:     "Ramai",
	constants.VehicleTrafficVeryBusy: "Sangat ramai (padat / sering macet)",
}

// trafficLines are the survey lines on vehicle traffic (prototype
// formatContext, 20-21 Oct), empty when no level was given.
func trafficLines(siteCtx *model.SiteContext) []string {
	if siteCtx.VehicleTrafficLevel == nil || *siteCtx.VehicleTrafficLevel == "" {
		return nil
	}
	label := trafficLabels[*siteCtx.VehicleTrafficLevel]
	if label == "" {
		label = *siteCtx.VehicleTrafficLevel
	}
	count := ""
	if n, m := siteCtx.VehicleTrafficCount, siteCtx.VehicleTrafficMinutes; n != nil && m != nil && *m > 0 {
		perHour := int(math.Round(float64(*n) * 60 / float64(*m)))
		count = fmt.Sprintf("; dihitung %d kendaraan dalam %d menit (± %d kendaraan/jam)", *n, *m, perHour)
	}
	lines := []string{fmt.Sprintf("- Lalu lintas kendaraan di jalan depan lokasi: %s%s", label, count)}
	if siteCtx.VehicleTrafficNote != nil && *siteCtx.VehicleTrafficNote != "" {
		lines = append(lines, fmt.Sprintf("    • Catatan lalu lintas: %s", *siteCtx.VehicleTrafficNote))
	}
	return lines
}

// competitorNoteOf is the staff's competitor note, "" when none.
func competitorNoteOf(siteCtx *model.SiteContext) string {
	if siteCtx == nil || siteCtx.CompetitorNote == nil {
		return ""
	}
	return *siteCtx.CompetitorNote
}

// ruleG: the staff's competitor note is unverified and never changes the
// competitor count (prototype rule G, 19 Oct).
const ruleG = `G) CATATAN KOMPETITOR STAF: catatan ini hanya keterangan Site Staff yang BELUM DIVERIFIKASI. JANGAN mengubah jumlah atau daftar kompetitor karenanya (kompetitor yang disebut "tutup/renovasi" tetap dihitung; kompetitor yang tidak ada di data peta tidak ditambahkan). Boleh disebut di narrative/risks/dataAssumptions, tetapi WAJIB ditulis sebagai catatan staf, mis. "Menurut catatan Site Staff (belum diverifikasi): …".`

// ruleH: how to score vehicle traffic (prototype rule H, 20 Oct).
const ruleH = `H) LALU LINTAS KENDARAAN (vehicleTraffic, label "Lalu Lintas Kendaraan"): nilai dari observasi staf (level + hitungan bila ada). footTraffic = pejalan kaki saja; kendaraan masuk vehicleTraffic, jangan dihitung dua kali.
   1. Lebih banyak kendaraan lewat = lebih terlihat dan lebih banyak calon pembeli yang mampir, TETAPI hanya jika kendaraan bisa berhenti: jika parkir "Tidak ada"/terbatas atau catatan menyebut jalan cepat/macet total/satu arah tanpa bahu jalan, skor vehicleTraffic TIDAK boleh tinggi dan sebut alasannya di factors.
   2. Patokan kasar: Sepi 40–55, Sedang 55–70, Ramai 70–85, Sangat ramai 65–85 (turunkan bila sulit berhenti). Hitungan per jam boleh menggeser ± 10.
   3. Jika level tidak diisi (data lama), estimasi dari jenis jalan/kota, tulis "estimasi" di factors, dan confidenceLevel maksimal "medium".
   4. Lalu lintas adalah SALAH SATU faktor; demografi, kompetitor, foot traffic, akses, dan ekonomi tetap penentu utama. Jangan membuat recommendation hanya karena lalu lintas.`

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
	lines = append(lines, trafficLines(siteCtx)...)
	if siteCtx.ParkingAvailability != nil && *siteCtx.ParkingAvailability != "" {
		label := parkingLabels[*siteCtx.ParkingAvailability]
		if label == "" {
			label = *siteCtx.ParkingAvailability
		}
		lines = append(lines, fmt.Sprintf("- Ketersediaan parkir: %s", label))
	}
	if siteCtx.ParkingNote != nil && *siteCtx.ParkingNote != "" {
		lines = append(lines, fmt.Sprintf("    • Catatan parkir: %s", *siteCtx.ParkingNote))
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
	if note := competitorNoteOf(siteCtx); note != "" {
		lines = append(lines, fmt.Sprintf(`- CATATAN KOMPETITOR DARI SITE STAFF (teks bebas, BELUM DIVERIFIKASI, lihat aturan G): "%s"`, note))
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
		(siteCtx.VehicleTrafficLevel != nil && *siteCtx.VehicleTrafficLevel != "") ||
		(siteCtx.ParkingNote != nil && *siteCtx.ParkingNote != "") ||
		competitorNoteOf(siteCtx) != "" ||
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
// With measured (local §6) it adds the DATA TERUKUR block after the user
// context and rule E; nil gives exactly the original prompt.
func buildPrompt(site *model.Site, siteCtx *model.SiteContext, competitors []*model.SiteCompetitor, notes []*model.SiteCustomNote, measured *dto.NearbyMeasured) string {
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
	if measured != nil {
		contextBlock += "\n" + strings.TrimSuffix(measuredBlock(measured), "\n")
		ctxRule += "\n\n" + measuredRule(measured)
	}
	// November scope (API-CHANGES §13.5, §6 note 19 Oct)
	if competitorNoteOf(siteCtx) != "" {
		ctxRule += "\n\n" + ruleG
	}
	ctxRule += "\n\n" + ruleH

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
   1. overallScore = %s. Tidak boleh menyimpang lebih dari 1 poin. (Lalu lintas kendaraan bobotnya lebih kecil: faktor utama, tapi tidak boleh mendominasi.)
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
		contextBlock, overallScoreFormula, site.SiteCity, site.SiteName1, ctxRule,
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

// GeminiMaxOutputTokens is the default answer budget. site-dev-api sent
// 8000, but on gemini-2.5-flash that budget also covers the model's
// thinking (3,000-4,300 tokens in the live check of 23 Oct), and 1 of 6
// real answers was cut off (MAX_TOKENS, no insight). 12000 leaves room; a
// normal answer still uses ~5,000-7,000 (billed by use, not by the cap).
const GeminiMaxOutputTokens = 12000

// GeminiRequest is the JSON body posted to generateContent for in (prompt +
// response schema, temperature 0.35, GeminiMaxOutputTokens).
func GeminiRequest(in GeminiInput) ([]byte, error) {
	return geminiRequest(in, GeminiMaxOutputTokens)
}

func geminiRequest(in GeminiInput, maxOutputTokens int) ([]byte, error) {
	body := geminiRequestBody{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: buildPrompt(in.Site, in.Context, in.Competitors, in.Notes, in.Measured)}}}},
	}
	body.GenerationConfig.ResponseMimeType = "application/json"
	body.GenerationConfig.ResponseSchema = responseSchema
	body.GenerationConfig.Temperature = 0.35
	body.GenerationConfig.MaxOutputTokens = maxOutputTokens

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
		Headline:         headlineOf(site, out),
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

// headlineOf is the fixed headline of the prompt ("<site name> — <rec>
// (<score>/100)", rule 3), built here: in the live check (23 Oct) Gemini's
// JSON mode sometimes garbled the dash (a literal "\u2014", or "\nA\n").
// Gemini's own text is kept when the site has no name (preview without
// one) or the answer has no recommendation.
func headlineOf(site *model.Site, out geminiPayload) string {
	if site == nil || strings.TrimSpace(site.SiteName1) == "" || out.Recommendation == "" {
		return out.Headline
	}
	return fmt.Sprintf("%s — %s (%d/100)", site.SiteName1, out.Recommendation, out.OverallScore)
}

// --- local §6: measured map data in the prompt (site-dev-api 2191e07
// nearby_area.go, from the prototype's server/gemini.ts) ---

// NearbySourceLabel is how a map data source is named on screen and in the
// prompt.
func NearbySourceLabel(source string) string {
	switch source {
	case constants.NearbySourceGoogle:
		return "Google Places"
	case constants.NearbySourceOSM:
		return "OpenStreetMap"
	case constants.NearbySourceMixed:
		return "Google Places + OpenStreetMap"
	}
	return source
}

// NearbyExamplesPerGroup is how many nearest examples per non-competitor
// category go to Gemini (counts stay complete).
const NearbyExamplesPerGroup = 5

var nearbyCategoryAI = map[string]string{
	constants.NearbyCategoryCompetitor: "Pesaing bermerek (minimarket dan kedai kopi)",
	constants.NearbyCategoryOffice:     "Kantor",
	constants.NearbyCategorySchool:     "Sekolah (TK sampai SMA/SMK)",
	constants.NearbyCategoryUniversity: "Kampus (universitas/akademi)",
	constants.NearbyCategoryApartment:  "Apartemen",
}

// fmtM: "120 m", "1,2 km".
func fmtM(m int) string {
	if m >= 1000 {
		return strings.Replace(fmt.Sprintf("%.1f km", float64(m)/1000), ".", ",", 1)
	}
	return fmt.Sprintf("%d m", m)
}

func containsString(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// formatMeasured renders the DATA TERUKUR lines.
func formatMeasured(m *dto.NearbyMeasured) string {
	src := NearbySourceLabel(m.Source)
	var lines []string
	for _, c := range constants.NearbyCategories {
		if _, ok := m.Counts[c]; !ok {
			continue // category switched off (places.categories)
		}
		var items []dto.NearbyMeasuredItem
		for _, x := range m.Items {
			if x.Category == c {
				items = append(items, x)
			}
		}
		capped := containsString(m.Capped, c)
		n := fmt.Sprintf("%d", m.Counts[c])
		if capped {
			n = "≥ " + n
		}
		cappedNote := ""
		if capped {
			cappedNote = " (sumber memotong hasil; bisa lebih)"
		}
		label := func(x dto.NearbyMeasuredItem) string {
			name := x.Name
			if name == "" {
				name = "(tanpa nama)"
			}
			prefix := ""
			if c == constants.NearbyCategoryCompetitor && x.Brand != "" && x.Brand != BrandOther && !strings.Contains(strings.ToLower(x.Name), strings.ToLower(x.Brand)) {
				prefix = x.Brand + " "
			}
			return fmt.Sprintf("%s%s %s", prefix, name, fmtM(x.DistanceM))
		}

		if c == constants.NearbyCategoryCompetitor {
			var list []string
			for i, x := range items {
				if i >= 25 {
					break
				}
				list = append(list, label(x))
			}
			more := ""
			if len(items) > 25 {
				more = fmt.Sprintf(", +%d lainnya", len(items)-25)
			}
			line := fmt.Sprintf("- %s: %s tercatat di %s", nearbyCategoryAI[c], n, src)
			if len(list) > 0 {
				line += " → " + strings.Join(list, "; ") + more
			}
			lines = append(lines, line+cappedNote)
			continue
		}

		per := m.ExamplesPerGroup
		if per <= 0 {
			per = NearbyExamplesPerGroup
		}
		var ex []string
		for i, x := range items {
			if i >= per {
				break
			}
			ex = append(ex, label(x))
		}
		line := fmt.Sprintf("- %s: %s tempat bernama tercatat di %s", nearbyCategoryAI[c], n, src)
		if len(ex) > 0 {
			line += " (contoh terdekat: " + strings.Join(ex, "; ") + ")"
		}
		lines = append(lines, line+cappedNote)
	}
	if fm := m.NearestFamilyMart; fm != nil {
		lines = append(lines, fmt.Sprintf("- FamilyMart terdekat (daftar toko FamilyMart): %s (kode %s), %s; FamilyMart dalam radius 1 km: %d", fm.Name, fm.Code, fmtM(fm.DistanceM), fm.Within1Km))
	}
	lines = append(lines, fmt.Sprintf(`Catatan: semua angka di atas = tempat BERNAMA yang tercatat di %s dalam radius %d m dari koordinat. Ini BUKAN sensus lengkap: 0 berarti "tidak tercatat di %s", bukan pasti tidak ada; tempat tanpa nama, bangunan baru, atau dalam pembangunan bisa belum tercatat.`, src, m.RadiusM, src))
	return strings.Join(lines, "\n")
}

// measuredBlock is the prompt section after the user context.
func measuredBlock(m *dto.NearbyMeasured) string {
	return fmt.Sprintf(`
============================
DATA TERUKUR (%s, radius %d m dari koordinat; diambil otomatis oleh sistem, BUKAN klaim user; sama dengan angka yang dilihat staf):
============================
%s
`, NearbySourceLabel(m.Source), m.RadiusM, formatMeasured(m))
}

// measuredRule is rule E of the prompt.
func measuredRule(m *dto.NearbyMeasured) string {
	src := NearbySourceLabel(m.Source)
	return fmt.Sprintf(`E) DATA TERUKUR (WAJIB DIPAKAI bila tersedia)
   1. Dasarkan skor footTraffic, competition, demographics, dan accessibility terutama pada DATA TERUKUR di atas, dikombinasikan dengan observasi user. Kantor, sekolah, kampus, dan apartemen adalah sumber pembeli (pekerja, pelajar/orang tua, mahasiswa, penghuni).
   2. Setiap angka tempat sekitar yang Anda sebut HARUS sama persis dengan DATA TERUKUR dan ditulis sebagai "tercatat di %[1]s" dengan radius, awalan "Data peta:" (contoh: "Data peta: 6 minimarket pesaing dan 12 kantor tercatat di %[1]s dalam %[2]d m, terdekat Indomaret 120 m"). Kutip minimal satu di metrics.competition.factors dan metrics.footTraffic.factors.
   3. JANGAN menyebut jumlah, nama, atau jarak tempat yang tidak ada di DATA TERUKUR atau di observasi user. Jangan memperkirakan "puluhan sekolah" dsb. JANGAN menulis angka ini sebagai jumlah pasti/sensus ("ada tepat 12 kantor"); tulis "tercatat".
   4. competitors: HANYA pesaing dari DATA TERUKUR (nama/merek & jarak persis, distanceKm = meter/1000) ditambah kompetitor teramati user; JANGAN mengarang pesaing lain.
   5. Jika klaim user bertentangan dengan DATA TERUKUR (mis. "tidak ada kompetitor" padahal data mencatat beberapa), tambahkan contextWarnings.
   6. Di dataAssumptions tulis: "Data tempat sekitar: tempat bernama yang tercatat di %[1]s, radius %[2]d m dari titik; bukan sensus lengkap (tempat tanpa nama/bangunan baru bisa belum tercatat)."
   7. Pertimbangkan jarak ke FamilyMart terdekat untuk risiko kanibalisasi (< 500 m = risiko tinggi).`, src, m.RadiusM)
}
