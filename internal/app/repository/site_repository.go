package repository

import (
	"context"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type ISiteRepository interface {
	// GetByIDs loads the sites for a set of SiteIDs (e.g. to add site
	// names to a task list).
	GetByIDs(ctx context.Context, siteIDs []string) (sites []*model.Site, err error)
	// GetIDsBySearch returns the SiteIDs whose ID, names or address lines
	// contain search (LIKE '%search%').
	GetIDsBySearch(ctx context.Context, search string) (siteIDs []string, err error)
	GetByID(ctx context.Context, siteID string) (site *model.Site, err error)
	// GetAttachmentsBySiteID returns a site's attachments, oldest first.
	GetAttachmentsBySiteID(ctx context.Context, siteID string) (attachments []*model.SiteAttachment, err error)
}

type SiteRepository struct {
	opt opt.Options
}

func NewSiteRepository() ISiteRepository {
	return &SiteRepository{
		opt: global.Setup.Common,
	}
}

func (r *SiteRepository) GetByIDs(ctx context.Context, siteIDs []string) (sites []*model.Site, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteRepository.GetByIDs")
	defer span.End()

	if len(siteIDs) == 0 {
		return
	}

	db := readDB(ctx, r.opt)

	if err = db.Where("SiteID IN ?", siteIDs).Find(&sites).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *SiteRepository) GetIDsBySearch(ctx context.Context, search string) (siteIDs []string, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteRepository.GetIDsBySearch")
	defer span.End()

	db := readDB(ctx, r.opt)

	like := "%" + search + "%"
	err = db.Model(new(model.Site)).
		Where("SiteID LIKE ? OR SiteName1 LIKE ? OR SiteName2 LIKE ? OR SiteAddress1 LIKE ? OR SiteAddress2 LIKE ? OR SiteAddress3 LIKE ?",
			like, like, like, like, like, like).
		Pluck("SiteID", &siteIDs).Error
	if err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *SiteRepository) GetByID(ctx context.Context, siteID string) (site *model.Site, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteRepository.GetByID")
	defer span.End()

	db := readDB(ctx, r.opt)

	site = &model.Site{}
	if err = db.Where("SiteID = ?", siteID).First(site).Error; err != nil {
		span.RecordError(err)
		site = nil
		return
	}

	return
}

func (r *SiteRepository) GetAttachmentsBySiteID(ctx context.Context, siteID string) (attachments []*model.SiteAttachment, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteRepository.GetAttachmentsBySiteID")
	defer span.End()

	db := readDB(ctx, r.opt)

	if err = db.Where("SiteID = ?", siteID).Order("SiteAttachmentID ASC").Find(&attachments).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}
