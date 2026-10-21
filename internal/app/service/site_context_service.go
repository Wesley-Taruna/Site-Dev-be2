package service

import (
	"context"

	"site-dev-backend/global"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type ISiteContextService interface {
	// GetBySiteID is GET /v1/site/:id/context (never filled in = 500
	// "record not found", as in site-dev-api).
	GetBySiteID(ctx context.Context, siteID string) (res dto.SiteContextResponseGet, err error)
}

type SiteContextService struct {
	opt                   opt.Options
	RepositorySiteContext repository.ISiteContextRepository
}

func NewSiteContextService() ISiteContextService {
	return &SiteContextService{
		opt:                   global.Setup.Common,
		RepositorySiteContext: repository.NewSiteContextRepository(),
	}
}

func (s *SiteContextService) GetBySiteID(ctx context.Context, siteID string) (res dto.SiteContextResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "SiteContextService.GetBySiteID")
	defer span.End()

	var siteContext *model.SiteContext
	siteContext, err = s.RepositorySiteContext.GetBySiteID(ctx, siteID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	var competitors []*model.SiteCompetitor
	competitors, err = s.RepositorySiteContext.GetCompetitorsBySiteID(ctx, siteID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	var notes []*model.SiteCustomNote
	notes, err = s.RepositorySiteContext.GetCustomNotesBySiteID(ctx, siteID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res = toSiteContextResponse(siteContext, competitors, notes)

	return
}
