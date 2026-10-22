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
	// Upsert creates the survey row or overwrites every field (a map
	// update, so a field can be emptied again).
	Upsert(ctx context.Context, siteContext *model.SiteContext) (err error)
	// ReplaceCompetitors / ReplaceCustomNotes delete the site's list and
	// insert the new one (the form is saved as a whole).
	ReplaceCompetitors(ctx context.Context, siteID string, competitors []*model.SiteCompetitor) (err error)
	ReplaceCustomNotes(ctx context.Context, siteID string, notes []*model.SiteCustomNote) (err error)
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

func (r *SiteContextRepository) Upsert(ctx context.Context, siteContext *model.SiteContext) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteContextRepository.Upsert")
	defer span.End()

	db := writeDB(ctx, r.opt)

	var count int64
	if err = db.Model(new(model.SiteContext)).Where("SiteID = ?", siteContext.SiteID).Count(&count).Error; err != nil {
		span.RecordError(err)
		return
	}
	if count == 0 {
		if err = db.Create(siteContext).Error; err != nil {
			span.RecordError(err)
		}
		return
	}

	updates := map[string]interface{}{
		"FootTrafficObservation": siteContext.FootTrafficObservation,
		"PeakHours":              siteContext.PeakHours,
		"ParkingAvailability":    siteContext.ParkingAvailability,
		"PublicTransport":        siteContext.PublicTransport,
		"AnchorLandmark":         siteContext.AnchorLandmark,
		"ObservedDemographics":   siteContext.ObservedDemographics,
		// November scope (API-CHANGES §13.5)
		"VehicleTrafficLevel":   siteContext.VehicleTrafficLevel,
		"VehicleTrafficCount":   siteContext.VehicleTrafficCount,
		"VehicleTrafficMinutes": siteContext.VehicleTrafficMinutes,
		"VehicleTrafficNote":    siteContext.VehicleTrafficNote,
		"ParkingNote":           siteContext.ParkingNote,
		"CompetitorNote":        siteContext.CompetitorNote,
		"UpdatedBy":             siteContext.UpdatedBy,
		"UpdatedAt":             siteContext.UpdatedAt,
	}
	if err = db.Model(new(model.SiteContext)).Where("SiteID = ?", siteContext.SiteID).Updates(updates).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *SiteContextRepository) ReplaceCompetitors(ctx context.Context, siteID string, competitors []*model.SiteCompetitor) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteContextRepository.ReplaceCompetitors")
	defer span.End()

	db := writeDB(ctx, r.opt)

	if err = db.Where("SiteID = ?", siteID).Delete(new(model.SiteCompetitor)).Error; err != nil {
		span.RecordError(err)
		return
	}
	if len(competitors) == 0 {
		return
	}
	if err = db.Create(&competitors).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *SiteContextRepository) ReplaceCustomNotes(ctx context.Context, siteID string, notes []*model.SiteCustomNote) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteContextRepository.ReplaceCustomNotes")
	defer span.End()

	db := writeDB(ctx, r.opt)

	if err = db.Where("SiteID = ?", siteID).Delete(new(model.SiteCustomNote)).Error; err != nil {
		span.RecordError(err)
		return
	}
	if len(notes) == 0 {
		return
	}
	if err = db.Create(&notes).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}
