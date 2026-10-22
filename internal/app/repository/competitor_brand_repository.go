package repository

import (
	"context"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

// ICompetitorBrandRepository reads CompetitorBrands (November scope,
// API-CHANGES §12; the brand admin endpoints come later).
type ICompetitorBrandRepository interface {
	// Get returns every brand in Sort order (then Slug).
	Get(ctx context.Context) (brands []*model.CompetitorBrand, err error)
}

type CompetitorBrandRepository struct {
	opt opt.Options
}

func NewCompetitorBrandRepository() ICompetitorBrandRepository {
	return &CompetitorBrandRepository{
		opt: global.Setup.Common,
	}
}

func (r *CompetitorBrandRepository) Get(ctx context.Context) (brands []*model.CompetitorBrand, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "CompetitorBrandRepository.Get")
	defer span.End()

	if err = readDB(ctx, r.opt).Order("Sort ASC, Slug ASC").Find(&brands).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}
