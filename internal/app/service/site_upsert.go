package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs/log"
)

// notesOrDefault keeps history notes from being empty.
func notesOrDefault(notes string, fallback string) string {
	if notes == "" {
		return fallback
	}
	return notes
}

func (s *SiteService) UpsertWithStatus(ctx context.Context, req dto.SiteRequestUpsert, status string, actorID int64) (res dto.SiteUpsertResponse, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "SiteService.UpsertWithStatus")
	defer span.End()

	res, err = s.upsertWithStatus(ctx, req, status, actorID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		res = dto.SiteUpsertResponse{}
		return
	}

	return
}

func (s *SiteService) upsertWithStatus(ctx context.Context, req dto.SiteRequestUpsert, status string, actorID int64) (res dto.SiteUpsertResponse, err error) {
	// the actor's name goes into the history notes (site-dev-api took it
	// from its token, plan Q1b)
	actor, err := getUser(ctx, s.RepositoryUser, actorID)
	if err != nil {
		return
	}
	actorName := actor.Username
	w := s.taskService.workflow
	isUpdate := req.Site.SiteID != ""

	var site model.Site
	if isUpdate {
		var existing *model.Site
		if existing, err = s.RepositorySite.GetByID(ctx, req.Site.SiteID); err != nil {
			err = fmt.Errorf("Site %s %w", req.Site.SiteID, err)
			return
		}
		site = *existing
	} else {
		if _, findErr := s.RepositorySite.GetByName1(ctx, req.Site.SiteName1); findErr == nil {
			err = echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Site with name %s already exists", req.Site.SiteName1))
			return
		} else if err = ignoreNotFound(findErr); err != nil {
			return
		}

		var siteID string
		if siteID, err = s.RepositoryRunningID.NextSiteID(ctx); err != nil {
			return
		}
		site = model.Site{SiteID: siteID, CreatedBy: actorID}
	}

	f := req.Site
	site.SiteName1 = f.SiteName1
	site.SiteName2 = f.SiteName2
	site.SiteAddress1 = f.SiteAddress1
	site.SiteAddress2 = f.SiteAddress2
	site.SiteAddress3 = f.SiteAddress3
	site.SitePostalCode = f.SitePostalCode
	site.SiteDistrict = f.SiteDistrict
	site.SiteCity = f.SiteCity
	site.SiteRegion = f.SiteRegion
	site.SiteLat = f.SiteLat
	site.SiteLon = f.SiteLon
	site.SiteTelepon = f.SiteTelepon
	site.SiteTeleponExt = f.SiteTeleponExt
	site.SiteMobilePhone = f.SiteMobilePhone
	site.SitePic = f.SitePic
	site.SiteFax = f.SiteFax
	site.SitePlannedOpeningDate = f.SitePlannedOpeningDate.Time
	site.SiteSalesArea = f.SiteSalesArea
	site.SiteSellingAlcohol = f.SiteSellingAlcohol
	site.SiteOpen24Hours = f.SiteOpen24Hours

	// UpdatedBy/UpdatedAt only on a real update; a new site keeps NULL
	if isUpdate {
		now := time.Now()
		updatedBy := actorID
		site.UpdatedBy = &updatedBy
		site.UpdatedAt = &now
	}

	var attachments []*model.SiteAttachment
	for _, a := range req.Attachment {
		attachments = append(attachments, &model.SiteAttachment{
			SiteID:    site.SiteID,
			FileName:  a.FileName,
			Path:      a.Path,
			CreatedAt: time.Now(),
		})
	}

	var siteContext *model.SiteContext
	var competitors []*model.SiteCompetitor
	var notes []*model.SiteCustomNote
	if req.Context != nil {
		siteContext, competitors, notes = siteContextRows(site.SiteID, *req.Context, actorID, time.Now())
	}

	// duplicate check (local §8): before the transaction, so a 409 holds no
	// transaction open; off by default
	var duplicates []dto.NearbyRequestResponseGet
	dupOutcome := duplicateNone
	if status == constants.TaskStatusSubmited && s.duplicateCheck.mode == constants.DuplicateCheckConfirm && !(site.SiteLat == 0 && site.SiteLon == 0) {
		if duplicates, err = s.taskService.nearbyActive(ctx, site.SiteLat, site.SiteLon, s.duplicateCheck.radiusM, site.SiteID); err != nil {
			return
		}
		dupOutcome = decideDuplicate(s.duplicateCheck.mode, len(duplicates), req.ConfirmDuplicate)
		if dupOutcome == duplicateBlock {
			err = &DuplicateNearbyError{RadiusM: s.duplicateCheck.radiusM, Items: duplicates}
			return
		}
	}

	var taskID string
	err = s.RepositoryTransaction.Do(ctx, func(ctx context.Context) error {
		if err := s.RepositorySite.Upsert(ctx, &site); err != nil {
			return err
		}

		if req.Attachment != nil {
			if err := s.RepositorySite.ReplaceAttachments(ctx, site.SiteID, attachments); err != nil {
				return err
			}
		}

		if siteContext != nil {
			if err := s.RepositorySiteContext.Upsert(ctx, siteContext); err != nil {
				return err
			}
			if err := s.RepositorySiteContext.ReplaceCompetitors(ctx, site.SiteID, competitors); err != nil {
				return err
			}
			if err := s.RepositorySiteContext.ReplaceCustomNotes(ctx, site.SiteID, notes); err != nil {
				return err
			}
		}

		existingTask, err := s.RepositoryTask.GetBySiteID(ctx, site.SiteID)
		taskFound := err == nil
		if err != nil {
			if err := ignoreNotFound(err); err != nil {
				return err
			}
		}

		if !taskFound {
			created, err := w.createTask(ctx, site.SiteID, actorID, notesOrDefault(req.Notes, fmt.Sprintf("Site baru dibuat oleh %s", actorName)))
			if err != nil {
				return err
			}
			taskID = created.TaskID
		} else {
			taskID = existingTask.TaskID
		}

		if status == constants.TaskStatusSubmited {
			if taskFound && site.CreatedBy != actorID {
				return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("Hanya pembuat site %s yang bisa submit", site.SiteName1))
			}
			if dupOutcome == duplicateConfirm {
				if err := w.recordHistory(ctx, taskID, constants.TaskActionDuplicateConfirmed, actorID, duplicateConfirmedNotes(s.duplicateCheck.radiusM, duplicates)); err != nil {
					return err
				}
			}
			// submitTask refuses a task that is not DRAFT/REVISION
			return w.submitTask(ctx, taskID, actorID, notesOrDefault(req.Notes, fmt.Sprintf("Site diajukan oleh %s untuk proses approval", actorName)))
		}

		// DRAFT: never move a task back once it was sent; REVISION stays
		// editable so the creator can fix and send again
		if taskFound {
			if existingTask.TaskStatus != constants.TaskStatusDraft && existingTask.TaskStatus != constants.TaskStatusRevision {
				return echo.NewHTTPError(http.StatusBadRequest, "Data site sudah disubmit tidak bisa diedit kembali")
			}
			if site.CreatedBy != actorID {
				return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("Hanya pembuat site %s yang bisa update", site.SiteName1))
			}
			return w.recordHistory(ctx, taskID, constants.TaskActionUpdate, actorID, notesOrDefault(req.Notes, fmt.Sprintf("Data site diperbarui oleh %s", actorName)))
		}
		return nil
	})
	if err != nil {
		return
	}

	s.maybeGenerateAIInsight(ctx, site.SiteID, status, isUpdate, actorID, req.Insight)

	var taskDetail dto.TaskDetailResponseGet
	if taskDetail, err = s.taskService.detail(ctx, taskID); err != nil {
		return
	}

	res = dto.SiteUpsertResponse{
		Site: toSiteResponse(&site),
		Task: taskDetail,
	}
	if siteContext != nil {
		contextRes := toSiteContextResponse(siteContext, competitors, notes)
		res.Context = &contextRes
	}
	if req.Attachment != nil {
		res.Attachments = make([]dto.AttachmentResponse, 0, len(attachments))
		for _, a := range attachments {
			res.Attachments = append(res.Attachments, dto.AttachmentResponse{
				SiteAttachmentID: a.SiteAttachmentID,
				FileName:         a.FileName,
				Path:             a.Path,
				CreatedAt:        a.CreatedAt,
			})
		}
	}
	return
}

// maybeGenerateAIInsight (best effort, never fails the save):
//   - an insight sent with the request (made by preview) is stored as is;
//   - a new site (DRAFT or SUBMITED): generate;
//   - an updated DRAFT without an insight yet: generate;
//   - otherwise nothing.
func (s *SiteService) maybeGenerateAIInsight(ctx context.Context, siteID string, status string, isUpdate bool, actorID int64, insight *dto.AIInsightResponseGet) {
	w := s.taskService.workflow
	if insight != nil {
		if err := w.ServicesAIInsight.Persist(ctx, siteID, *insight, actorID); err != nil {
			w.logError(fmt.Errorf("site_service: failed to persist client-provided AI insight for site %s: %w", siteID, err))
		}
		return
	}

	needGenerate := !isUpdate
	if isUpdate && status == constants.TaskStatusDraft {
		_, err := s.RepositorySiteAIInsight.GetLatestBySiteID(ctx, siteID)
		needGenerate = err != nil
	}
	if !needGenerate {
		return
	}
	w.enqueueAIInsight(ctx, siteID, actorID)
}
