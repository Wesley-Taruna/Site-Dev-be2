package repository

import (
	"context"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type ISiteAIInsightRepository interface {
	// Replace deletes the site's insight (the database cascades to its child
	// rows) and inserts insight; insight.SiteAIID is set afterwards.
	Replace(ctx context.Context, insight *model.SiteAIInsight) (err error)
	// GetLatestBySiteID returns the site's insight (gorm ErrRecordNotFound
	// if none was generated).
	GetLatestBySiteID(ctx context.Context, siteID string) (insight *model.SiteAIInsight, err error)

	CreateMetrics(ctx context.Context, metrics []*model.SiteAIMetric) (err error)
	CreateRisks(ctx context.Context, risks []*model.SiteAIRisk) (err error)
	CreateOpportunities(ctx context.Context, opportunities []*model.SiteAIOpportunity) (err error)
	CreateDataAssumptions(ctx context.Context, assumptions []*model.SiteAIDataAssumption) (err error)
	CreateContextWarnings(ctx context.Context, warnings []*model.SiteAIContextWarning) (err error)

	GetMetrics(ctx context.Context, siteAIID int64) (metrics []*model.SiteAIMetric, err error)
	GetRisks(ctx context.Context, siteAIID int64) (risks []*model.SiteAIRisk, err error)
	GetOpportunities(ctx context.Context, siteAIID int64) (opportunities []*model.SiteAIOpportunity, err error)
	GetDataAssumptions(ctx context.Context, siteAIID int64) (assumptions []*model.SiteAIDataAssumption, err error)
	GetContextWarnings(ctx context.Context, siteAIID int64) (warnings []*model.SiteAIContextWarning, err error)
}

type SiteAIInsightRepository struct {
	opt opt.Options
}

func NewSiteAIInsightRepository() ISiteAIInsightRepository {
	return &SiteAIInsightRepository{
		opt: global.Setup.Common,
	}
}

func (r *SiteAIInsightRepository) Replace(ctx context.Context, insight *model.SiteAIInsight) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteAIInsightRepository.Replace")
	defer span.End()

	db := writeDB(ctx, r.opt)

	if err = db.Where("SiteID = ?", insight.SiteID).Delete(new(model.SiteAIInsight)).Error; err != nil {
		span.RecordError(err)
		return
	}
	if err = db.Create(insight).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *SiteAIInsightRepository) GetLatestBySiteID(ctx context.Context, siteID string) (insight *model.SiteAIInsight, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteAIInsightRepository.GetLatestBySiteID")
	defer span.End()

	db := readDB(ctx, r.opt)

	insight = &model.SiteAIInsight{}
	if err = db.Where("SiteID = ?", siteID).Order("GeneratedAt DESC").First(insight).Error; err != nil {
		span.RecordError(err)
		insight = nil
		return
	}

	return
}

// createAll inserts rows in one statement; nothing to insert = no query.
func createAll[T any](ctx context.Context, o opt.Options, name string, rows []*T) (err error) {
	ctx, span := o.Trace.Start(ctx, name)
	defer span.End()

	if len(rows) == 0 {
		return
	}
	if err = writeDB(ctx, o).Create(&rows).Error; err != nil {
		span.RecordError(err)
		return
	}
	return
}

// childRows reads the rows of one insight, oldest first.
func childRows[T any](ctx context.Context, o opt.Options, name string, siteAIID int64) (rows []*T, err error) {
	ctx, span := o.Trace.Start(ctx, name)
	defer span.End()

	if err = readDB(ctx, o).Where("SiteAIID = ?", siteAIID).Order("ID ASC").Find(&rows).Error; err != nil {
		span.RecordError(err)
		return
	}
	return
}

func (r *SiteAIInsightRepository) CreateMetrics(ctx context.Context, metrics []*model.SiteAIMetric) (err error) {
	return createAll(ctx, r.opt, "SiteAIInsightRepository.CreateMetrics", metrics)
}

func (r *SiteAIInsightRepository) CreateRisks(ctx context.Context, risks []*model.SiteAIRisk) (err error) {
	return createAll(ctx, r.opt, "SiteAIInsightRepository.CreateRisks", risks)
}

func (r *SiteAIInsightRepository) CreateOpportunities(ctx context.Context, opportunities []*model.SiteAIOpportunity) (err error) {
	return createAll(ctx, r.opt, "SiteAIInsightRepository.CreateOpportunities", opportunities)
}

func (r *SiteAIInsightRepository) CreateDataAssumptions(ctx context.Context, assumptions []*model.SiteAIDataAssumption) (err error) {
	return createAll(ctx, r.opt, "SiteAIInsightRepository.CreateDataAssumptions", assumptions)
}

func (r *SiteAIInsightRepository) CreateContextWarnings(ctx context.Context, warnings []*model.SiteAIContextWarning) (err error) {
	return createAll(ctx, r.opt, "SiteAIInsightRepository.CreateContextWarnings", warnings)
}

func (r *SiteAIInsightRepository) GetMetrics(ctx context.Context, siteAIID int64) (metrics []*model.SiteAIMetric, err error) {
	return childRows[model.SiteAIMetric](ctx, r.opt, "SiteAIInsightRepository.GetMetrics", siteAIID)
}

func (r *SiteAIInsightRepository) GetRisks(ctx context.Context, siteAIID int64) (risks []*model.SiteAIRisk, err error) {
	return childRows[model.SiteAIRisk](ctx, r.opt, "SiteAIInsightRepository.GetRisks", siteAIID)
}

func (r *SiteAIInsightRepository) GetOpportunities(ctx context.Context, siteAIID int64) (opportunities []*model.SiteAIOpportunity, err error) {
	return childRows[model.SiteAIOpportunity](ctx, r.opt, "SiteAIInsightRepository.GetOpportunities", siteAIID)
}

func (r *SiteAIInsightRepository) GetDataAssumptions(ctx context.Context, siteAIID int64) (assumptions []*model.SiteAIDataAssumption, err error) {
	return childRows[model.SiteAIDataAssumption](ctx, r.opt, "SiteAIInsightRepository.GetDataAssumptions", siteAIID)
}

func (r *SiteAIInsightRepository) GetContextWarnings(ctx context.Context, siteAIID int64) (warnings []*model.SiteAIContextWarning, err error) {
	return childRows[model.SiteAIContextWarning](ctx, r.opt, "SiteAIInsightRepository.GetContextWarnings", siteAIID)
}
