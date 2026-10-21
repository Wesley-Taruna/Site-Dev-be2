package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type ISiteContextService interface {
	// GetBySiteID is GET /v1/site/:id/context (never filled in = 500
	// "record not found", as in site-dev-api).
	GetBySiteID(ctx context.Context, siteID string) (res dto.SiteContextResponseGet, err error)
	// Update is PUT /v1/task/context/:site_id: replaces the survey. Only the
	// site's creator, only while the task is DRAFT or REVISION.
	Update(ctx context.Context, siteID string, req dto.SiteContextRequestUpsert, userID int64) (res dto.SiteContextResponseGet, err error)
}

type SiteContextService struct {
	opt                   opt.Options
	RepositorySiteContext repository.ISiteContextRepository
	RepositorySite        repository.ISiteRepository
	RepositoryTask        repository.ITaskRepository
	RepositoryTaskHistory repository.ITaskHistoryRepository
	RepositoryUser        repository.IUserRepository
	RepositoryTransaction repository.ITransactionRepository
}

func NewSiteContextService() ISiteContextService {
	return &SiteContextService{
		opt:                   global.Setup.Common,
		RepositorySiteContext: repository.NewSiteContextRepository(),
		RepositorySite:        repository.NewSiteRepository(),
		RepositoryTask:        repository.NewTaskRepository(),
		RepositoryTaskHistory: repository.NewTaskHistoryRepository(),
		RepositoryUser:        repository.NewUserRepository(),
		RepositoryTransaction: repository.NewTransactionRepository(),
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

func (s *SiteContextService) Update(ctx context.Context, siteID string, req dto.SiteContextRequestUpsert, userID int64) (res dto.SiteContextResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "SiteContextService.Update")
	defer span.End()

	res, err = s.update(ctx, siteID, req, userID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		res = dto.SiteContextResponseGet{}
		return
	}

	return
}

func (s *SiteContextService) update(ctx context.Context, siteID string, req dto.SiteContextRequestUpsert, userID int64) (res dto.SiteContextResponseGet, err error) {
	// the actor's name goes into the history note (site-dev-api took it
	// from its token, plan Q1b)
	actor, err := getUser(ctx, s.RepositoryUser, userID)
	if err != nil {
		return
	}

	site, err := s.RepositorySite.GetByID(ctx, siteID)
	if err != nil {
		err = fmt.Errorf("site %s: %w", siteID, err)
		return
	}
	if site.CreatedBy != userID {
		err = echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("Hanya pembuat site %s yang bisa edit konteks", site.SiteName1))
		return
	}

	task, err := s.RepositoryTask.GetBySiteID(ctx, siteID)
	if err != nil {
		err = fmt.Errorf("task for site %s: %w", siteID, err)
		return
	}
	if task.TaskStatus != constants.TaskStatusDraft && task.TaskStatus != constants.TaskStatusRevision {
		err = echo.NewHTTPError(http.StatusBadRequest, "Task sudah disubmit, konteks tidak bisa diedit lagi")
		return
	}

	now := time.Now()
	row, competitors, notes := siteContextRows(siteID, req, userID, now)

	err = s.RepositoryTransaction.Do(ctx, func(ctx context.Context) error {
		if err := s.RepositorySiteContext.Upsert(ctx, row); err != nil {
			return err
		}
		if err := s.RepositorySiteContext.ReplaceCompetitors(ctx, siteID, competitors); err != nil {
			return err
		}
		if err := s.RepositorySiteContext.ReplaceCustomNotes(ctx, siteID, notes); err != nil {
			return err
		}
		return s.RepositoryTaskHistory.Create(ctx, newTaskHistory(task.TaskID, constants.TaskActionUpdate, userID,
			historyMeta{role: actor.RoleID, round: roundOf(task)}, fmt.Sprintf("Konteks on-site diperbarui oleh %s", actor.Username)))
	})
	if err != nil {
		return
	}

	res = toSiteContextResponse(row, competitors, notes)
	return
}

// siteContextRows turns the survey form into the rows to save.
func siteContextRows(siteID string, req dto.SiteContextRequestUpsert, userID int64, now time.Time) (row *model.SiteContext, competitors []*model.SiteCompetitor, notes []*model.SiteCustomNote) {
	row = &model.SiteContext{
		SiteID:                 siteID,
		FootTrafficObservation: req.FootTrafficObservation,
		PeakHours:              req.PeakHours,
		ParkingAvailability:    req.ParkingAvailability,
		PublicTransport:        req.PublicTransport,
		AnchorLandmark:         req.AnchorLandmark,
		ObservedDemographics:   req.ObservedDemographics,
		CreatedBy:              userID,
		UpdatedBy:              userID,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	for _, c := range req.Competitors {
		competitors = append(competitors, &model.SiteCompetitor{
			SiteID:         siteID,
			CompetitorName: c.CompetitorName,
			CompetitorType: c.CompetitorType,
			Distance:       c.Distance,
			CreatedAt:      now,
		})
	}
	for _, n := range req.CustomNotes {
		notes = append(notes, &model.SiteCustomNote{
			SiteID:    siteID,
			Title:     n.Title,
			Content:   n.Content,
			CreatedAt: now,
		})
	}
	return
}
