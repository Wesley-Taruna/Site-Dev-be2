package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gorm.io/gorm"
)

// approvalHistoryMessage is the history note of an APPROVE/REJECT/REVISION
// decision (site-dev-api dto.BuildApprovalHistoryMessage).
func approvalHistoryMessage(decision string, username string, roleName string, notes string) string {
	actor := username
	if roleName != "" {
		actor = fmt.Sprintf("%s (%s)", username, roleName)
	}
	if decision == constants.TaskActionReject {
		return fmt.Sprintf("Site di reject oleh %s dengan catatan (%s)", actor, notes)
	}
	if decision == constants.TaskActionRevision {
		return fmt.Sprintf("Site diminta revisi oleh %s dengan catatan (%s)", actor, notes)
	}
	// November scope (API-CHANGES §13.1); wording is our choice
	if decision == constants.TaskActionHold {
		return fmt.Sprintf("Site ditahan (Hold Status) oleh %s dengan alasan (%s)", actor, notes)
	}
	if decision == constants.TaskActionRelease {
		if notes == "" {
			return fmt.Sprintf("Hold Status dilepas oleh %s", actor)
		}
		return fmt.Sprintf("Hold Status dilepas oleh %s dengan catatan (%s)", actor, notes)
	}
	if notes == "" {
		return fmt.Sprintf("Site di approve oleh %s", actor)
	}
	return fmt.Sprintf("Site di approve oleh %s dengan catatan (%s)", actor, notes)
}

// approverLevel finds the level the caller decides on: PENDING for every
// decision, HOLD for RELEASE (November scope, API-CHANGES §13.1: a held
// level only takes RELEASE; RELEASE needs a held level).
func (s *TaskService) approverLevel(ctx context.Context, taskID string, headerID uint, actorID int64, decision string) (*model.ApprovalDetail, error) {
	if decision == constants.TaskActionRelease {
		detail, err := s.RepositoryApproval.GetHeldDetailByApprover(ctx, headerID, actorID)
		if err == nil {
			return detail, nil
		}
		if _, pendingErr := s.RepositoryApproval.GetPendingDetailByApprover(ctx, headerID, actorID); pendingErr == nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Pengajuan ini tidak sedang dalam Hold Status.")
		}
		return nil, fmt.Errorf("task_service: no pending approval for this user on task %s: %w", taskID, err)
	}

	detail, err := s.RepositoryApproval.GetPendingDetailByApprover(ctx, headerID, actorID)
	if err == nil {
		return detail, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if _, heldErr := s.RepositoryApproval.GetHeldDetailByApprover(ctx, headerID, actorID); heldErr == nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Pengajuan ini sedang Anda tahan. Tekan Lanjutkan dulu.")
		}
	}
	return nil, fmt.Errorf("task_service: no pending approval for this user on task %s: %w", taskID, err)
}

func (s *TaskService) Approve(ctx context.Context, taskID string, actorID int64, req dto.TaskActionRequest) (err error) {
	ctx, span := s.opt.Trace.Start(ctx, "TaskService.Approve")
	defer span.End()

	if err = s.approve(ctx, taskID, actorID, req); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	return
}

func (s *TaskService) approve(ctx context.Context, taskID string, actorID int64, req dto.TaskActionRequest) error {
	w := s.workflow

	task, err := s.RepositoryTask.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	header, err := s.RepositoryApproval.GetHeaderByTaskID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task_service: approval header not found for task %s: %w", taskID, err)
	}
	if header.OverallStatus != constants.ApprovalOverallStatusPending {
		return fmt.Errorf("task_service: task %s is not awaiting approval (status=%s)", taskID, task.TaskStatus)
	}

	detail, err := s.approverLevel(ctx, taskID, header.ID, actorID, req.Decision)
	if err != nil {
		return err
	}

	var actorUsername string
	if actor, userErr := s.RepositoryUser.GetByID(ctx, actorID); userErr == nil {
		actorUsername = actor.Username
	}

	// history: the role of the level decided on, in the task's round
	meta := historyMeta{role: detail.ApproverRoleID, round: roundOf(task)}

	var notifyNext *model.ApprovalDetail
	err = s.RepositoryTransaction.Do(ctx, func(ctx context.Context) error {
		now := time.Now()

		if req.Decision == constants.TaskActionRelease {
			return s.release(ctx, w, taskID, actorID, detail, req.Notes, actorUsername, meta, now)
		}

		detailUpdates := map[string]interface{}{
			"Notes":     req.Notes,
			"ActedAt":   now,
			"UpdatedAt": now,
		}
		switch req.Decision {
		case constants.TaskActionApprove:
			detailUpdates["Status"] = constants.ApprovalDetailStatusApproved
		case constants.TaskActionReject:
			detailUpdates["Status"] = constants.ApprovalDetailStatusRejected
		case constants.TaskActionRevision:
			detailUpdates["Status"] = constants.ApprovalDetailStatusRevision
		case constants.TaskActionHold:
			detailUpdates["Status"] = constants.ApprovalDetailStatusHold
		}

		rows, err := s.RepositoryApproval.UpdateDetailIfPending(ctx, detail.ID, detailUpdates)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("task_service: approval already acted on for task %s (conflict)", taskID)
		}

		historyAction := constants.TaskActionApprove
		if req.Decision == constants.TaskActionReject {
			historyAction = constants.TaskActionReject
		} else if req.Decision == constants.TaskActionRevision {
			historyAction = constants.TaskActionRevision
		} else if req.Decision == constants.TaskActionHold {
			historyAction = constants.TaskActionHold
		}
		historyNotes := approvalHistoryMessage(req.Decision, actorUsername, w.roleNameOf(ctx, detail.ApproverRoleID), req.Notes)
		if err := w.recordHistory(ctx, taskID, historyAction, actorID, meta, historyNotes); err != nil {
			return err
		}

		if req.Decision == constants.TaskActionReject {
			if err := s.RepositoryApproval.UpdateHeader(ctx, header.ID, map[string]interface{}{
				"OverallStatus": constants.ApprovalOverallStatusRejected,
				"UpdatedAt":     now,
			}); err != nil {
				return err
			}
			return s.RepositoryTask.UpdateStatus(ctx, taskID, map[string]interface{}{
				"TaskStatus": constants.TaskStatusRejected,
				"UpdatedBy":  actorID,
				"UpdatedAt":  now,
			})
		}

		if req.Decision == constants.TaskActionHold {
			// the round stays PENDING; no e-mail (open: review §5 #9)
			return s.RepositoryTask.UpdateStatus(ctx, taskID, map[string]interface{}{
				"TaskStatus": constants.TaskStatusHold(detail.ApproverRoleID),
				"UpdatedBy":  actorID,
				"UpdatedAt":  now,
			})
		}

		if req.Decision == constants.TaskActionRevision {
			// the round stays PENDING; the next submit opens a new one
			return s.RepositoryTask.UpdateStatus(ctx, taskID, map[string]interface{}{
				"TaskStatus": constants.TaskStatusRevision,
				"UpdatedBy":  actorID,
				"UpdatedAt":  now,
			})
		}

		// APPROVE at the last level: done
		if detail.Level >= header.TotalLevel {
			if err := s.RepositoryApproval.UpdateHeader(ctx, header.ID, map[string]interface{}{
				"OverallStatus": constants.ApprovalOverallStatusApproved,
				"UpdatedAt":     now,
			}); err != nil {
				return err
			}
			if err := s.RepositoryTask.UpdateStatus(ctx, taskID, map[string]interface{}{
				"TaskStatus": constants.TaskStatusCompleted,
				"UpdatedBy":  actorID,
				"UpdatedAt":  now,
			}); err != nil {
				return err
			}
			return w.recordHistory(ctx, taskID, constants.TaskActionComplete, actorID, meta, "Task Complete")
		}

		// next level = the smallest level above this one (levels come from
		// Roles.RoleLevel and need not be consecutive)
		details, err := s.RepositoryApproval.GetDetailsByHeaderID(ctx, header.ID)
		if err != nil {
			return err
		}
		var next *model.ApprovalDetail
		for _, d := range details {
			if d.Level > detail.Level && (next == nil || d.Level < next.Level) {
				next = d
			}
		}
		if next == nil {
			return fmt.Errorf("task_service: no next approval step found after level %d for task %s", detail.Level, taskID)
		}
		if err := s.RepositoryApproval.UpdateDetailByID(ctx, next.ID, map[string]interface{}{
			"Status":    constants.ApprovalDetailStatusPending,
			"UpdatedAt": now,
		}); err != nil {
			return err
		}
		notifyNext = next

		if err := s.RepositoryTask.UpdateStatus(ctx, taskID, map[string]interface{}{
			"TaskStatus": constants.TaskStatusWaiting(next.ApproverRoleID),
			"UpdatedBy":  actorID,
			"UpdatedAt":  now,
		}); err != nil {
			return err
		}

		forwardNotes := fmt.Sprintf("Approval diteruskan ke %s, status PENDING", w.roleNameOf(ctx, next.ApproverRoleID))
		if err := w.recordHistory(ctx, taskID, constants.TaskActionForward, actorID, meta, forwardNotes); err != nil {
			return err
		}

		return s.RepositoryApproval.UpdateHeader(ctx, header.ID, map[string]interface{}{
			"CurrentLevel": next.Level,
			"UpdatedAt":    now,
		})
	})
	if err != nil {
		return err
	}
	if notifyNext != nil {
		w.notifyApprovalPending(ctx, taskID, task.SiteID, header.SubmittedBy, notifyNext, req.Notes)
	}
	return nil
}

// release takes a held level back to PENDING (acted_at and reason cleared,
// as the prototype does) and the task back to WAITING_<ROLE>; then the
// approver decides normally. No e-mail: the approver was told already.
func (s *TaskService) release(ctx context.Context, w *taskWorkflow, taskID string, actorID int64, detail *model.ApprovalDetail, notes string, actorUsername string, meta historyMeta, now time.Time) error {
	rows, err := s.RepositoryApproval.UpdateDetailIfHeld(ctx, detail.ID, map[string]interface{}{
		"Status":    constants.ApprovalDetailStatusPending,
		"Notes":     "",
		"ActedAt":   nil,
		"UpdatedAt": now,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("task_service: approval already acted on for task %s (conflict)", taskID)
	}

	historyNotes := approvalHistoryMessage(constants.TaskActionRelease, actorUsername, w.roleNameOf(ctx, detail.ApproverRoleID), notes)
	if err := w.recordHistory(ctx, taskID, constants.TaskActionRelease, actorID, meta, historyNotes); err != nil {
		return err
	}

	return s.RepositoryTask.UpdateStatus(ctx, taskID, map[string]interface{}{
		"TaskStatus": constants.TaskStatusWaiting(detail.ApproverRoleID),
		"UpdatedBy":  actorID,
		"UpdatedAt":  now,
	})
}
