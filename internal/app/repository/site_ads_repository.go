package repository

import (
	"context"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

// ISiteAdsRepository reads and writes SiteAds (November scope, portal).
type ISiteAdsRepository interface {
	// GetBySiteIDs loads the ADS rows of these sites (sites without one are
	// missing).
	GetBySiteIDs(ctx context.Context, siteIDs []string) (ads []*model.SiteAds, err error)
	// Upsert creates the row or overwrites value, note, who and when.
	Upsert(ctx context.Context, ads *model.SiteAds) (err error)
}

type SiteAdsRepository struct {
	opt opt.Options
}

func NewSiteAdsRepository() ISiteAdsRepository {
	return &SiteAdsRepository{
		opt: global.Setup.Common,
	}
}

func (r *SiteAdsRepository) GetBySiteIDs(ctx context.Context, siteIDs []string) (ads []*model.SiteAds, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteAdsRepository.GetBySiteIDs")
	defer span.End()

	if len(siteIDs) == 0 {
		return
	}
	if err = readDB(ctx, r.opt).Where("SiteID IN (?)", siteIDs).Find(&ads).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *SiteAdsRepository) Upsert(ctx context.Context, ads *model.SiteAds) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteAdsRepository.Upsert")
	defer span.End()

	db := writeDB(ctx, r.opt)

	tx := db.Model(new(model.SiteAds)).Where("SiteID = ?", ads.SiteID).Updates(map[string]interface{}{
		"AdsValue":  ads.AdsValue,
		"AdsNote":   ads.AdsNote,
		"UpdatedBy": ads.UpdatedBy,
		"UpdatedAt": ads.UpdatedAt,
	})
	if err = tx.Error; err != nil {
		span.RecordError(err)
		return
	}
	if tx.RowsAffected > 0 {
		return
	}
	if err = db.Create(ads).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}
