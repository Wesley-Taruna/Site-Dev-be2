package repository

import (
	"context"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type ISiteContextRepository interface {
	// GetBySiteID returns the on-site survey of a site (gorm
	// ErrRecordNotFound if it was never filled in).
	GetBySiteID(ctx context.Context, siteID string) (siteContext *model.SiteContext, err error)
	// GetCompetitorsBySiteID / GetCustomNotesBySiteID return the survey
	// lists, oldest first.
	GetCompetitorsBySiteID(ctx context.Context, siteID string) (competitors []*model.SiteCompetitor, err error)
	GetCustomNotesBySiteID(ctx context.Context, siteID string) (notes []*model.SiteCustomNote, err error)
}

type SiteContextRepository struct {
	opt opt.Options
}

func NewSiteContextRepository() ISiteContextRepository {
	return &SiteContextRepository{
		opt: global.Setup.Common,
	}
}

func (r *SiteContextRepository) GetBySiteID(ctx context.Context, siteID string) (siteContext *model.SiteContext, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteContextRepository.GetBySiteID")
	defer span.End()

	db := readDB(ctx, r.opt)

	siteContext = &model.SiteContext{}
	if err = db.Where("SiteID = ?", siteID).First(siteContext).Error; err != nil {
		span.RecordError(err)
		siteContext = nil
		return
	}

	return
}

func (r *SiteContextRepository) GetCompetitorsBySiteID(ctx context.Context, siteID string) (competitors []*model.SiteCompetitor, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteContextRepository.GetCompetitorsBySiteID")
	defer span.End()

	db := readDB(ctx, r.opt)

	if err = db.Where("SiteID = ?", siteID).Order("ID ASC").Find(&competitors).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *SiteContextRepository) GetCustomNotesBySiteID(ctx context.Context, siteID string) (notes []*model.SiteCustomNote, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteContextRepository.GetCustomNotesBySiteID")
	defer span.End()

	db := readDB(ctx, r.opt)

	if err = db.Where("SiteID = ?", siteID).Order("ID ASC").Find(&notes).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}
