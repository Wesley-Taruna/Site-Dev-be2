package service

import (
	"context"
	"fmt"
	"time"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/log"
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
	if notes == "" {
		return fmt.Sprintf("Site di approve oleh %s", actor)
	}
	return fmt.Sprintf("Site di approve oleh %s dengan catatan (%s)", actor, notes)
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

	detail, err := s.RepositoryApproval.GetPendingDetailByApprover(ctx, header.ID, actorID)
	if err != nil {
		return fmt.Errorf("task_service: no pending approval for this user on task %s: %w", taskID, err)
	}

	var actorUsername string
	if actor, userErr := s.RepositoryUser.GetByID(ctx, actorID); userErr == nil {
		actorUsername = actor.Username
	}

	var notifyNext *model.ApprovalDetail
	err = s.RepositoryTransaction.Do(ctx, func(ctx context.Context) error {
		now := time.Now()

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
		}
		historyNotes := approvalHistoryMessage(req.Decision, actorUsername, w.roleNameOf(ctx, detail.ApproverRoleID), req.Notes)
		if err := w.recordHistory(ctx, taskID, historyAction, actorID, historyNotes); err != nil {
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
			return w.recordHistory(ctx, taskID, constants.TaskActionComplete, actorID, "Task Complete")
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
		if err := w.recordHistory(ctx, taskID, constants.TaskActionForward, actorID, forwardNotes); err != nil {
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
