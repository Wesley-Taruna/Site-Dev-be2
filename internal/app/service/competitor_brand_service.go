package service

import (
	"context"
	"encoding/json"
	"fmt"

	"site-dev-backend/global"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type ICompetitorBrandService interface {
	// Get is GET /v1/competitor-brands: every brand (inactive ones too,
	// with active = false; the app shows and matches only active ones), in
	// Sort order. Any logged-in user.
	Get(ctx context.Context) (res []dto.CompetitorBrandResponseGet, err error)
}

type CompetitorBrandService struct {
	opt                       opt.Options
	RepositoryCompetitorBrand repository.ICompetitorBrandRepository
}

func NewCompetitorBrandService() ICompetitorBrandService {
	return &CompetitorBrandService{
		opt:                       global.Setup.Common,
		RepositoryCompetitorBrand: repository.NewCompetitorBrandRepository(),
	}
}

func (s *CompetitorBrandService) Get(ctx context.Context) (res []dto.CompetitorBrandResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "CompetitorBrandService.Get")
	defer span.End()

	var brands []*model.CompetitorBrand
	brands, err = s.RepositoryCompetitorBrand.Get(ctx)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res = make([]dto.CompetitorBrandResponseGet, 0, len(brands))
	for _, b := range brands {
		res = append(res, dto.CompetitorBrandResponseGet{
			Slug:        b.Slug,
			Name:        b.Name,
			Patterns:    s.patternsOf(b),
			LogoURL:     b.LogoPath,
			Color:       b.Color,
			Type:        b.Type,
			LegacyType:  b.LegacyType,
			Active:      b.Active,
			Placeholder: b.IsPlaceholder,
			Sort:        b.Sort,
		})
	}

	return
}

// patternsOf reads the JSON array; a broken value gives [] (logged), so
// one bad row never breaks the list.
func (s *CompetitorBrandService) patternsOf(b *model.CompetitorBrand) []string {
	patterns := []string{}
	if b.Patterns == "" {
		return patterns
	}
	if err := json.Unmarshal([]byte(b.Patterns), &patterns); err != nil {
		s.opt.Logger.AddMessage(log.ErrorLevel, fmt.Errorf("competitor brand %s: patterns are not a JSON list: %w", b.Slug, err)).Print()
		return []string{}
	}
	return patterns
}
