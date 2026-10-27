package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gorm.io/gorm"
)

// Wes, 26 Oct: the staff close their own requests in two ways.
//   - "Batalkan pengajuan" (POST /v1/task/:id/cancel): a request returned
//     to them (REVISION) becomes CANCELLED; kept with its history, hidden
//     from the active lists, not active for the duplicate check.
//   - "Hapus draft" (DELETE /v1/site/:id): a draft that was never sent is
//     removed for good (site, survey, photos, AI result, task, history).
//     The AI limits keep what the draft used (Redis counts untouched).

// Answers of the two staff actions (Indonesian, shown by the app).
const (
	msgTaskNotFound     = "Pengajuan tidak ditemukan."
	msgCancelNotOwner   = "Hanya pembuat pengajuan yang bisa membatalkannya."
	msgCancelWrongState = "Hanya pengajuan yang Ditolak (bisa direvisi) yang bisa dibatalkan."
	msgDeleteNotOwner   = "Hanya pembuat draft yang bisa menghapusnya."
	msgDeleteWrongState = "Hanya draft yang belum pernah dikirim yang bisa dihapus."
	msgNotesTooLong     = "Catatan maksimal 500 karakter."
)

// taskByAnyID reads a task by task_id ("TS…") or site_id ("SD…"/"ST…"); not
// found = 404.
func (s *TaskService) taskByAnyID(ctx context.Context, id string) (task *model.Task, err error) {
	switch {
	case strings.HasPrefix(id, constants.TaskIDPrefix):
		task, err = s.RepositoryTask.GetByID(ctx, id)
	case constants.IsSiteID(id):
		task, err = s.RepositoryTask.GetBySiteID(ctx, id)
	default:
		err = gorm.ErrRecordNotFound
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = echo.NewHTTPError(http.StatusNotFound, msgTaskNotFound)
	}
	return
}

// Cancel is POST /v1/task/:id/cancel (id = task_id or site_id).
func (s *TaskService) Cancel(ctx context.Context, id string, actorID int64, req dto.TaskCancelRequest) (err error) {
	ctx, span := s.opt.Trace.Start(ctx, "TaskService.Cancel")
	defer span.End()

	if err = s.cancel(ctx, id, actorID, req); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	return
}

func (s *TaskService) cancel(ctx context.Context, id string, actorID int64, req dto.TaskCancelRequest) error {
	notes := strings.TrimSpace(req.Notes)
	if utf8.RuneCountInString(notes) > constants.NotesMax {
		return echo.NewHTTPError(http.StatusBadRequest, msgNotesTooLong)
	}

	actor, err := getUser(ctx, s.RepositoryUser, actorID)
	if err != nil {
		return err
	}
	task, err := s.taskByAnyID(ctx, id)
	if err != nil {
		return err
	}
	if task.CreatedBy != actorID {
		return echo.NewHTTPError(http.StatusForbidden, msgCancelNotOwner)
	}
	if task.TaskStatus != constants.TaskStatusRevision {
		return echo.NewHTTPError(http.StatusConflict, msgCancelWrongState)
	}

	w := s.workflow
	meta := historyMeta{role: actor.RoleID, round: roundOf(task)}
	return s.RepositoryTransaction.Do(ctx, func(ctx context.Context) error {
		now := time.Now()
		rows, err := s.RepositoryTask.UpdateStatusIf(ctx, task.TaskID, constants.TaskStatusRevision, map[string]interface{}{
			"TaskStatus": constants.TaskStatusCancelled,
			"UpdatedBy":  actorID,
			"UpdatedAt":  now,
		})
		if err != nil {
			return err
		}
		if rows == 0 {
			return echo.NewHTTPError(http.StatusConflict, msgCancelWrongState)
		}
		// the round of the REVISION stays as the approver left it (PENDING
		// header, REVISION level): nothing reads it once the task is closed
		historyNotes := approvalHistoryMessage(constants.TaskActionCancel, actor.Username, w.roleNameOf(ctx, actor.RoleID), notes)
		return w.recordHistory(ctx, task.TaskID, constants.TaskActionCancel, actorID, meta, historyNotes)
	})
}

// DeleteDraft is DELETE /v1/site/:id (id = site_id or task_id).
func (s *SiteService) DeleteDraft(ctx context.Context, id string, actorID int64) (err error) {
	ctx, span := s.opt.Trace.Start(ctx, "SiteService.DeleteDraft")
	defer span.End()

	if err = s.deleteDraft(ctx, id, actorID); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	return
}

func (s *SiteService) deleteDraft(ctx context.Context, id string, actorID int64) error {
	ts := s.taskService
	if _, err := getUser(ctx, s.RepositoryUser, actorID); err != nil {
		return err
	}
	task, err := ts.taskByAnyID(ctx, id)
	if err != nil {
		return err
	}
	site, err := s.RepositorySite.GetByID(ctx, task.SiteID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, msgTaskNotFound)
	}
	if err != nil {
		return err
	}
	if site.CreatedBy != actorID || task.CreatedBy != actorID {
		return echo.NewHTTPError(http.StatusForbidden, msgDeleteNotOwner)
	}
	if task.TaskStatus != constants.TaskStatusDraft {
		return echo.NewHTTPError(http.StatusConflict, msgDeleteWrongState)
	}
	// never sent: no approval round and no SUBMIT/RESUBMIT in the history
	if _, headerErr := ts.RepositoryApproval.GetHeaderByTaskID(ctx, task.TaskID); headerErr == nil {
		return echo.NewHTTPError(http.StatusConflict, msgDeleteWrongState)
	} else if err := ignoreNotFound(headerErr); err != nil {
		return err
	}
	histories, err := ts.RepositoryTaskHistory.GetByTaskID(ctx, task.TaskID)
	if err != nil {
		return err
	}
	for _, h := range histories {
		if h.Action == constants.TaskActionSubmit || h.Action == constants.TaskActionResubmit {
			return echo.NewHTTPError(http.StatusConflict, msgDeleteWrongState)
		}
	}

	attachments, err := s.RepositorySite.GetAttachmentsBySiteID(ctx, site.SiteID)
	if err != nil {
		return err
	}

	err = s.RepositoryTransaction.Do(ctx, func(ctx context.Context) error {
		rows, err := s.RepositoryTask.UpdateStatusIf(ctx, task.TaskID, constants.TaskStatusDraft, map[string]interface{}{"UpdatedBy": actorID})
		if err != nil {
			return err
		}
		if rows == 0 { // sent meanwhile
			return echo.NewHTTPError(http.StatusConflict, msgDeleteWrongState)
		}
		return s.RepositoryTask.DeleteDraft(ctx, site.SiteID, task.TaskID)
	})
	if err != nil {
		return err
	}

	// the photo files, best effort after the commit (the rows are gone)
	for _, a := range attachments {
		bucket, object, splitErr := splitObjectPath(a.Path)
		if splitErr == nil {
			splitErr = s.ClientStorage.Delete(ctx, bucket, object)
		}
		if splitErr != nil {
			s.opt.Logger.AddMessage(log.ErrorLevel, fmt.Errorf("site_service: draft %s deleted, photo file %q not removed: %w", site.SiteID, a.Path, splitErr)).Print()
		}
	}
	return nil
}
