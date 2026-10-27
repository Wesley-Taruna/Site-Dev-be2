package client

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
)

// Overall score from ALL inputs (Wes, 27 Oct): the server computes it from
// the 6 metric scores, the risks' severities and Gemini's opportunity
// bonus (constants.ScoreRiskPenalty...), so the same answer always gives
// the same number and a wrong sum by the model cannot slip through. The
// recommendation follows the score band (prompt rule B2) and the headline
// is rebuilt.

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// metricBand is rule B3: ≥85 high, 70-84 good, 50-69 moderate, <50 low.
func metricBand(score int) string {
	switch {
	case score >= 85:
		return "high"
	case score >= 70:
		return "good"
	case score >= 50:
		return "moderate"
	}
	return "low"
}

// Recommendation is rule B2 for an overall score.
func Recommendation(score int) string {
	switch {
	case score >= 82:
		return "Strongly Recommended"
	case score >= 70:
		return "Recommended"
	case score >= 55:
		return "Proceed With Caution"
	}
	return "Reconsider"
}

// riskPenalty is the sum over risks (high 4, moderate 2, low 1), at most
// ScoreRiskPenaltyMax.
func riskPenalty(risks []dto.AIInsightRisk) int {
	p := 0
	for _, r := range risks {
		switch strings.ToLower(strings.TrimSpace(r.Severity)) {
		case "high":
			p += constants.ScoreRiskPenaltyHigh
		case "moderate":
			p += constants.ScoreRiskPenaltyModerate
		case "low":
			p += constants.ScoreRiskPenaltyLow
		}
	}
	return clampInt(p, 0, constants.ScoreRiskPenaltyMax)
}

// baseScore = round(0.18 × 5 main metrics + 0.10 × vehicleTraffic); an
// insight without vehicleTraffic (older shape) = average of the 5.
func baseScore(m dto.AIInsightMetrics) int {
	main := float64(m.FootTraffic.Score + m.Demographics.Score + m.Competition.Score + m.Accessibility.Score + m.Economic.Score)
	if m.VehicleTraffic == nil {
		return int(math.Round(main / 5))
	}
	return int(math.Round(0.18*main + 0.10*float64(m.VehicleTraffic.Score)))
}

// ScoreInsight checks and completes an insight: metric scores clamped to
// 0..100 with their band, overallScore = clamp(base − risk penalty +
// opportunity bonus, 0, 100), the recommendation from that score, the
// breakdown attached. opportunityBonus is clamped to 0..5.
func ScoreInsight(ins *dto.AIInsightResponseGet, opportunityBonus int) {
	for _, key := range constants.MetricKeyOrder {
		if m := metricOf(&ins.Metrics, key); m != nil {
			m.Score = clampInt(m.Score, 0, 100)
			m.Band = metricBand(m.Score)
		}
	}
	b := &dto.AIScoreBreakdown{
		Base:             baseScore(ins.Metrics),
		RiskPenalty:      riskPenalty(ins.Risks),
		OpportunityBonus: clampInt(opportunityBonus, 0, constants.ScoreOpportunityBonusMax),
	}
	ins.ScoreBreakdown = b
	ins.OverallScore = clampInt(b.Base-b.RiskPenalty+b.OpportunityBonus, 0, 100)
	ins.Recommendation = Recommendation(ins.OverallScore)
	ins.ScoreReason = fixReasonScore(strings.TrimSpace(ins.ScoreReason), ins.OverallScore)
}

// reasonTotal finds a total score written in the reason ("Skor total 64",
// "skor keseluruhan 91", "skor akhir: 70/100").
var reasonTotal = regexp.MustCompile(`(?i)(skor\s+(?:total|keseluruhan|akhir)\s*(?:adalah|sebesar|yaitu)?\s*:?\s*)(\d{1,3})(\s*/\s*100)?`)

// fixReasonScore: the prompt asks Gemini not to write the total (the
// server computes it), but in the live runs (27 Oct) it often did; a
// number there that differs from the server's is replaced by it.
func fixReasonScore(reason string, score int) string {
	return reasonTotal.ReplaceAllStringFunc(reason, func(m string) string {
		p := reasonTotal.FindStringSubmatch(m)
		return p[1] + fmt.Sprint(score) + p[3]
	})
}

// RescoreSentInsight re-checks an insight a client sends back (preview
// answer stored on save): only one with a breakdown (made by the new
// scoring) is recomputed, so the stored score always matches its metrics,
// risks and bonus; older ones stay as sent. The headline keeps the site
// name it was made with ("<name> — <rec> (<score>/100)").
func RescoreSentInsight(ins *dto.AIInsightResponseGet) {
	if ins.ScoreBreakdown == nil {
		return
	}
	ScoreInsight(ins, ins.ScoreBreakdown.OpportunityBonus)
	if name, _, ok := strings.Cut(ins.Headline, " — "); ok && strings.TrimSpace(name) != "" {
		ins.Headline = scoredHeadline(&model.Site{SiteName1: name}, ins)
	}
}

// scoredHeadline is the fixed headline "<site name> — <rec> (<score>/100)",
// or the insight's own when the site has no name.
func scoredHeadline(site *model.Site, ins *dto.AIInsightResponseGet) string {
	if site == nil || strings.TrimSpace(site.SiteName1) == "" {
		return ins.Headline
	}
	return fmt.Sprintf("%s — %s (%d/100)", site.SiteName1, ins.Recommendation, ins.OverallScore)
}

// metricOf points at one of the metrics (nil for vehicleTraffic while the
// insight has none).
func metricOf(m *dto.AIInsightMetrics, key string) *dto.AIInsightMetric {
	switch key {
	case constants.MetricKeyFootTraffic:
		return &m.FootTraffic
	case constants.MetricKeyDemographics:
		return &m.Demographics
	case constants.MetricKeyCompetition:
		return &m.Competition
	case constants.MetricKeyAccessibility:
		return &m.Accessibility
	case constants.MetricKeyEconomic:
		return &m.Economic
	case constants.MetricKeyVehicleTraffic:
		return m.VehicleTraffic
	}
	return nil
}
