package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/client"
	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

// factorsSep joins a metric's factors in SiteAIMetrics.Factors (plain text).
const factorsSep = ";"

type IAIInsightService interface {
	// Generate is POST /v1/task/ai-insight (and the queue job): counts one
	// from the caller's daily quota, asks Gemini, replaces the site's
	// insight.
	Generate(ctx context.Context, siteID string, actorID int64) (res dto.AIInsightResponseGet, err error)
	// Get is GET /v1/task/ai-insight/:site_id: the stored insight, with the
	// competitors from the survey.
	Get(ctx context.Context, siteID string) (res dto.AIInsightResponseGet, err error)
	// Preview is POST /v1/task/ai-insight/preview: Gemini on data not saved
	// yet; nothing is stored. Counts one from the quota.
	Preview(ctx context.Context, req dto.AIInsightRequestPreview, actorID int64) (res dto.AIInsightResponseGet, err error)
	// Persist stores an insight made earlier (preview) for siteID, without
	// calling Gemini.
	Persist(ctx context.Context, siteID string, insight dto.AIInsightResponseGet, actorID int64) (err error)
	// GetQuota is GET /v1/task/ai-insight/quota (does not use any).
	GetQuota(ctx context.Context, actorID int64) (res dto.AIInsightQuotaResponse, err error)
	// FlushQuota is the 00:00 queue job: every user's AI counter is reset.
	FlushQuota(ctx context.Context) (deleted int, err error)
}

type AIInsightService struct {
	opt                     opt.Options
	RepositorySiteAIInsight repository.ISiteAIInsightRepository
	RepositorySite          repository.ISiteRepository
	RepositorySiteContext   repository.ISiteContextRepository
	RepositoryQuota         repository.IQuotaRepository
	RepositoryTransaction   repository.ITransactionRepository
	ClientGemini            client.IGeminiClient
}

func NewAIInsightService() IAIInsightService {
	return &AIInsightService{
		opt:                     global.Setup.Common,
		RepositorySiteAIInsight: repository.NewSiteAIInsightRepository(),
		RepositorySite:          repository.NewSiteRepository(),
		RepositorySiteContext:   repository.NewSiteContextRepository(),
		RepositoryQuota:         repository.NewQuotaRepository(),
		RepositoryTransaction:   repository.NewTransactionRepository(),
		ClientGemini:            global.Setup.Clients.Gemini,
	}
}

func (s *AIInsightService) Generate(ctx context.Context, siteID string, actorID int64) (res dto.AIInsightResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "AIInsightService.Generate")
	defer span.End()

	res, err = s.generate(ctx, siteID, actorID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		res = dto.AIInsightResponseGet{}
		return
	}

	return
}

func (s *AIInsightService) generate(ctx context.Context, siteID string, actorID int64) (res dto.AIInsightResponseGet, err error) {
	if err = s.useQuota(ctx, actorID); err != nil {
		return
	}

	site, err := s.RepositorySite.GetByID(ctx, siteID)
	if err != nil {
		err = fmt.Errorf("site %s: %w", siteID, err)
		return
	}

	in := client.GeminiInput{Site: site}
	in.Context, err = s.RepositorySiteContext.GetBySiteID(ctx, siteID)
	if err != nil {
		// no survey yet is fine
		if err = ignoreNotFound(err); err != nil {
			return
		}
		in.Context = nil
	}
	if in.Context != nil {
		if in.Competitors, err = s.RepositorySiteContext.GetCompetitorsBySiteID(ctx, siteID); err != nil {
			return
		}
		if in.Notes, err = s.RepositorySiteContext.GetCustomNotesBySiteID(ctx, siteID); err != nil {
			return
		}
	}

	insight, geminiModel, err := s.ClientGemini.GenerateInsight(ctx, in)
	if err != nil {
		return
	}

	var row *model.SiteAIInsight
	if row, err = s.persist(ctx, &insight, geminiModel, actorID); err != nil {
		return
	}

	insight.SiteAIID = row.SiteAIID
	res = insight
	return
}

func (s *AIInsightService) Persist(ctx context.Context, siteID string, insight dto.AIInsightResponseGet, actorID int64) (err error) {
	ctx, span := s.opt.Trace.Start(ctx, "AIInsightService.Persist")
	defer span.End()

	insight.SiteID = siteID
	if _, err = s.persist(ctx, &insight, "", actorID); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	return
}

// persist replaces the site's insight row and writes its child rows in one
// transaction.
func (s *AIInsightService) persist(ctx context.Context, insight *dto.AIInsightResponseGet, geminiModel string, actorID int64) (row *model.SiteAIInsight, err error) {
	row = &model.SiteAIInsight{
		SiteID:           insight.SiteID,
		OverallScore:     insight.OverallScore,
		Recommendation:   insight.Recommendation,
		Headline:         insight.Headline,
		Narrative:        insight.Narrative,
		ConfidenceLevel:  insight.ConfidenceLevel,
		ConfidenceReason: insight.ConfidenceReason,
		GeminiModel:      geminiModel,
		GeneratedBy:      actorID,
		GeneratedAt:      insight.GeneratedAt,
	}

	err = s.RepositoryTransaction.Do(ctx, func(ctx context.Context) error {
		if err := s.RepositorySiteAIInsight.Replace(ctx, row); err != nil {
			return err
		}
		if err := s.RepositorySiteAIInsight.CreateMetrics(ctx, metricRows(row.SiteAIID, insight.Metrics)); err != nil {
			return err
		}
		risks := make([]*model.SiteAIRisk, 0, len(insight.Risks))
		for _, r := range insight.Risks {
			risks = append(risks, &model.SiteAIRisk{SiteAIID: row.SiteAIID, Title: r.Title, Severity: r.Severity, Mitigation: r.Mitigation})
		}
		if err := s.RepositorySiteAIInsight.CreateRisks(ctx, risks); err != nil {
			return err
		}
		opportunities := make([]*model.SiteAIOpportunity, 0, len(insight.Opportunities))
		for _, o := range insight.Opportunities {
			opportunities = append(opportunities, &model.SiteAIOpportunity{SiteAIID: row.SiteAIID, Opportunity: o})
		}
		if err := s.RepositorySiteAIInsight.CreateOpportunities(ctx, opportunities); err != nil {
			return err
		}
		assumptions := make([]*model.SiteAIDataAssumption, 0, len(insight.DataAssumptions))
		for _, a := range insight.DataAssumptions {
			assumptions = append(assumptions, &model.SiteAIDataAssumption{SiteAIID: row.SiteAIID, Assumption: a})
		}
		if err := s.RepositorySiteAIInsight.CreateDataAssumptions(ctx, assumptions); err != nil {
			return err
		}
		warnings := make([]*model.SiteAIContextWarning, 0, len(insight.ContextWarnings))
		for _, w := range insight.ContextWarnings {
			warnings = append(warnings, &model.SiteAIContextWarning{SiteAIID: row.SiteAIID, Field: w.Field, UserClaim: w.UserClaim,
				AIExpectation: w.AIExpectation, Severity: w.Severity, Message: w.Message})
		}
		return s.RepositorySiteAIInsight.CreateContextWarnings(ctx, warnings)
	})
	if err != nil {
		row = nil
	}
	return
}

// metricByKey points at one of the 5 metrics.
func metricByKey(m *dto.AIInsightMetrics, key string) *dto.AIInsightMetric {
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
	}
	return nil
}

// metricRows is the 5 SiteAIMetrics rows of an insight, in fixed order.
func metricRows(siteAIID int64, m dto.AIInsightMetrics) []*model.SiteAIMetric {
	rows := make([]*model.SiteAIMetric, 0, len(constants.MetricKeyOrder))
	for _, key := range constants.MetricKeyOrder {
		src := metricByKey(&m, key)
		rows = append(rows, &model.SiteAIMetric{
			SiteAIID:  siteAIID,
			MetricKey: key,
			Label:     src.Label,
			Score:     src.Score,
			Band:      src.Band,
			Summary:   src.Summary,
			Factors:   strings.Join(src.Factors, factorsSep),
		})
	}
	return rows
}

func (s *AIInsightService) Get(ctx context.Context, siteID string) (res dto.AIInsightResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "AIInsightService.Get")
	defer span.End()

	res, err = s.get(ctx, siteID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		res = dto.AIInsightResponseGet{}
		return
	}

	return
}

func (s *AIInsightService) get(ctx context.Context, siteID string) (res dto.AIInsightResponseGet, err error) {
	insight, err := s.RepositorySiteAIInsight.GetLatestBySiteID(ctx, siteID)
	if err != nil {
		err = fmt.Errorf("insight for site %s: %w", siteID, err)
		return
	}

	metrics, err := s.RepositorySiteAIInsight.GetMetrics(ctx, insight.SiteAIID)
	if err != nil {
		return
	}
	risks, err := s.RepositorySiteAIInsight.GetRisks(ctx, insight.SiteAIID)
	if err != nil {
		return
	}
	opportunities, err := s.RepositorySiteAIInsight.GetOpportunities(ctx, insight.SiteAIID)
	if err != nil {
		return
	}
	assumptions, err := s.RepositorySiteAIInsight.GetDataAssumptions(ctx, insight.SiteAIID)
	if err != nil {
		return
	}
	warnings, err := s.RepositorySiteAIInsight.GetContextWarnings(ctx, insight.SiteAIID)
	if err != nil {
		return
	}
	competitors, err := s.RepositorySiteContext.GetCompetitorsBySiteID(ctx, siteID)
	if err != nil {
		if err = ignoreNotFound(err); err != nil {
			return
		}
		competitors = nil
	}

	res = dto.AIInsightResponseGet{
		SiteAIID:         insight.SiteAIID,
		SiteID:           insight.SiteID,
		OverallScore:     insight.OverallScore,
		Recommendation:   insight.Recommendation,
		Headline:         insight.Headline,
		Narrative:        insight.Narrative,
		ConfidenceLevel:  insight.ConfidenceLevel,
		ConfidenceReason: insight.ConfidenceReason,
		GeneratedAt:      insight.GeneratedAt,
		Competitors:      make([]dto.AIInsightCompetitor, 0, len(competitors)),
		Risks:            make([]dto.AIInsightRisk, 0, len(risks)),
		Opportunities:    make([]string, 0, len(opportunities)),
		DataAssumptions:  make([]string, 0, len(assumptions)),
		ContextWarnings:  make([]dto.AIInsightContextWarning, 0, len(warnings)),
	}
	for _, m := range metrics {
		if dst := metricByKey(&res.Metrics, m.MetricKey); dst != nil {
			*dst = dto.AIInsightMetric{Label: m.Label, Score: m.Score, Band: m.Band, Summary: m.Summary, Factors: strings.Split(m.Factors, factorsSep)}
		}
	}
	for _, c := range competitors {
		res.Competitors = append(res.Competitors, dto.AIInsightCompetitor{Name: c.CompetitorName, DistanceKm: parseDistanceKm(c.Distance), Type: c.CompetitorType})
	}
	for _, r := range risks {
		res.Risks = append(res.Risks, dto.AIInsightRisk{Title: r.Title, Severity: r.Severity, Mitigation: r.Mitigation})
	}
	for _, o := range opportunities {
		res.Opportunities = append(res.Opportunities, o.Opportunity)
	}
	for _, a := range assumptions {
		res.DataAssumptions = append(res.DataAssumptions, a.Assumption)
	}
	for _, w := range warnings {
		res.ContextWarnings = append(res.ContextWarnings, dto.AIInsightContextWarning{Field: w.Field, UserClaim: w.UserClaim,
			AIExpectation: w.AIExpectation, Severity: w.Severity, Message: w.Message})
	}
	return
}

// parseDistanceKm reads the free-text survey distance ("500m", "2km", "2")
// as km; unreadable = 0.
func parseDistanceKm(s string) float64 {
	s = strings.ToLower(strings.TrimSpace(s))
	var numStr strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			numStr.WriteRune(r)
		} else if numStr.Len() > 0 {
			break
		}
	}
	n, err := strconv.ParseFloat(numStr.String(), 64)
	if err != nil {
		return 0
	}
	if strings.Contains(s, "km") {
		return n
	}
	if strings.Contains(s, "m") {
		return n / 1000
	}
	return n
}

func (s *AIInsightService) Preview(ctx context.Context, req dto.AIInsightRequestPreview, actorID int64) (res dto.AIInsightResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "AIInsightService.Preview")
	defer span.End()

	if err = s.useQuota(ctx, actorID); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	in := client.GeminiInput{Site: &model.Site{
		SiteName1:              req.Site.SiteName1,
		SiteName2:              req.Site.SiteName2,
		SiteAddress1:           req.Site.SiteAddress1,
		SiteAddress2:           req.Site.SiteAddress2,
		SiteAddress3:           req.Site.SiteAddress3,
		SitePostalCode:         req.Site.SitePostalCode,
		SiteDistrict:           req.Site.SiteDistrict,
		SiteCity:               req.Site.SiteCity,
		SiteRegion:             req.Site.SiteRegion,
		SiteLat:                req.Site.SiteLat,
		SiteLon:                req.Site.SiteLon,
		SitePlannedOpeningDate: req.Site.SitePlannedOpeningDate.Time,
		SiteSalesArea:          req.Site.SiteSalesArea,
		SiteSellingAlcohol:     req.Site.SiteSellingAlcohol,
		SiteOpen24Hours:        req.Site.SiteOpen24Hours,
	}}
	if c := req.Context; c != nil {
		in.Context = &model.SiteContext{
			FootTrafficObservation: c.FootTrafficObservation,
			PeakHours:              c.PeakHours,
			ParkingAvailability:    c.ParkingAvailability,
			PublicTransport:        c.PublicTransport,
			AnchorLandmark:         c.AnchorLandmark,
			ObservedDemographics:   c.ObservedDemographics,
		}
		for _, comp := range c.Competitors {
			in.Competitors = append(in.Competitors, &model.SiteCompetitor{CompetitorName: comp.CompetitorName, CompetitorType: comp.CompetitorType, Distance: comp.Distance})
		}
		for _, n := range c.CustomNotes {
			in.Notes = append(in.Notes, &model.SiteCustomNote{Title: n.Title, Content: n.Content})
		}
	}

	res, _, err = s.ClientGemini.GenerateInsight(ctx, in)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		res = dto.AIInsightResponseGet{}
		return
	}

	return
}

func (s *AIInsightService) GetQuota(ctx context.Context, actorID int64) (res dto.AIInsightQuotaResponse, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "AIInsightService.GetQuota")
	defer span.End()

	if res.Remaining, err = s.RepositoryQuota.Remaining(ctx, constants.QuotaAIInsight, actorID); err == nil {
		res.Limit, err = s.RepositoryQuota.Limit(ctx, constants.QuotaAIInsight)
	}
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		res = dto.AIInsightQuotaResponse{}
		return
	}

	return
}

func (s *AIInsightService) FlushQuota(ctx context.Context) (deleted int, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "AIInsightService.FlushQuota")
	defer span.End()

	if deleted, err = s.RepositoryQuota.FlushAll(ctx, constants.QuotaAIInsight); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	return
}

// useQuota counts one Gemini call; used up = 429 "Kuota generate AI insight
// harian (<limit>) sudah habis, coba lagi besok."
func (s *AIInsightService) useQuota(ctx context.Context, actorID int64) error {
	_, err := s.RepositoryQuota.Consume(ctx, constants.QuotaAIInsight, actorID)
	if errors.Is(err, repository.ErrAIInsightQuotaExceeded) {
		limit, limitErr := s.RepositoryQuota.Limit(ctx, constants.QuotaAIInsight)
		if limitErr != nil {
			return limitErr
		}
		return echo.NewHTTPError(http.StatusTooManyRequests, fmt.Sprintf("Kuota generate AI insight harian (%d) sudah habis, coba lagi besok.", limit))
	}
	return err
}
