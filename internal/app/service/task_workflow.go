package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/client"
	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

// taskWorkflow holds the steps of a task's life that the save/send
// (SiteService) and the approval (TaskService) share: create, send for
// approval, history, approval e-mail, AI insight job. They run inside the
// caller's transaction where there is one (site-dev-api's TaskService
// createTx/submitTx/recordHistory/notifyApprovalPending/enqueueAIInsightTask).
type taskWorkflow struct {
	opt                      opt.Options
	RepositoryTask           repository.ITaskRepository
	RepositoryTaskHistory    repository.ITaskHistoryRepository
	RepositoryApproval       repository.IApprovalRepository
	RepositoryApprovalMatrix repository.IApprovalMatrixRepository
	RepositoryApprovalCode   repository.IApprovalCodeRepository
	RepositoryRunningID      repository.IRunningIDRepository
	RepositoryUser           repository.IUserRepository
	RepositoryRole           repository.IRoleRepository
	RepositorySite           repository.ISiteRepository
	ClientTaskQueue          client.ITaskQueueClient
	ClientMail               client.IMailClient
	ServicesAIInsight        IAIInsightService
}

func newTaskWorkflow() *taskWorkflow {
	return &taskWorkflow{
		opt:                      global.Setup.Common,
		RepositoryTask:           repository.NewTaskRepository(),
		RepositoryTaskHistory:    repository.NewTaskHistoryRepository(),
		RepositoryApproval:       repository.NewApprovalRepository(),
		RepositoryApprovalMatrix: repository.NewApprovalMatrixRepository(),
		RepositoryApprovalCode:   repository.NewApprovalCodeRepository(),
		RepositoryRunningID:      repository.NewRunningIDRepository(),
		RepositoryUser:           repository.NewUserRepository(),
		RepositoryRole:           repository.NewRoleRepository(),
		RepositorySite:           repository.NewSiteRepository(),
		ClientTaskQueue:          global.Setup.Clients.TaskQueue,
		ClientMail:               global.Setup.Clients.Mail,
		ServicesAIInsight:        NewAIInsightService(),
	}
}

// logError logs a best-effort failure (the request itself goes on).
func (w *taskWorkflow) logError(err error) {
	w.opt.Logger.AddMessage(log.ErrorLevel, err).Print()
}

// roleNameOf is the role's display name, or the RoleID when unknown.
func (w *taskWorkflow) roleNameOf(ctx context.Context, roleID string) string {
	role, err := w.RepositoryRole.GetByID(ctx, roleID)
	if err != nil || role == nil {
		return roleID
	}
	return role.RoleName
}

// historyMeta is the actor's role at that moment and the submit round of a
// history row (November scope, API-CHANGES §13.3).
type historyMeta struct {
	role  string
	round int
}

// roundOf is a task's current submit round (1 when unknown).
func roundOf(task *model.Task) int {
	if task == nil || task.SubmitCount < 1 {
		return 1
	}
	return task.SubmitCount
}

// recordHistory appends one history row.
func (w *taskWorkflow) recordHistory(ctx context.Context, taskID string, action string, actorID int64, meta historyMeta, notes string) error {
	return w.RepositoryTaskHistory.Create(ctx, newTaskHistory(taskID, action, actorID, meta, notes))
}

// newTaskHistory builds a history row; an empty role or round 0 stays NULL.
func newTaskHistory(taskID string, action string, actorID int64, meta historyMeta, notes string) *model.TaskHistory {
	h := &model.TaskHistory{
		TaskID:  taskID,
		Action:  action,
		ActorID: actorID,
		Notes:   notes,
	}
	if meta.role != "" {
		role := meta.role
		h.ActorRole = &role
	}
	if meta.round > 0 {
		round := meta.round
		h.Round = &round
	}
	return h
}

// createTask makes the DRAFT task of a new site (submit round 1), with its
// CREATE history.
func (w *taskWorkflow) createTask(ctx context.Context, siteID string, createdBy int64, creatorRole string, notes string) (*model.Task, error) {
	taskID, err := w.RepositoryRunningID.NextTaskID(ctx)
	if err != nil {
		return nil, err
	}

	task := &model.Task{
		TaskID:      taskID,
		SiteID:      siteID,
		TaskStatus:  constants.TaskStatusDraft,
		CreatedBy:   createdBy,
		SubmitCount: 1,
	}
	if err := w.RepositoryTask.Create(ctx, task); err != nil {
		return nil, err
	}
	if err := w.recordHistory(ctx, taskID, constants.TaskActionCreate, createdBy, historyMeta{role: creatorRole, round: 1}, notes); err != nil {
		return nil, err
	}
	return task, nil
}

// submitTask sends a DRAFT or REVISION task for approval: a new approval
// round with one level per active approval matrix row (by role level; the
// first PENDING, the rest WAITING), the e-mail to the first approver, task
// status WAITING_<first role>, SUBMIT history. From REVISION (November
// scope, API-CHANGES §13.3) it is a resubmit: history RESUBMIT and
// SubmitCount + 1 (the new round starts at the first level again).
func (w *taskWorkflow) submitTask(ctx context.Context, taskID string, submittedBy int64, submitterRole string, notes string) error {
	task, err := w.RepositoryTask.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task.TaskStatus != constants.TaskStatusDraft && task.TaskStatus != constants.TaskStatusRevision {
		return fmt.Errorf("task_service: task %s cannot be submitted from status %s", taskID, task.TaskStatus)
	}

	chain, err := w.RepositoryApprovalMatrix.GetActiveChain(ctx)
	if err != nil {
		return fmt.Errorf("task_service: failed to resolve approval chain: %w", err)
	}
	if len(chain) == 0 {
		return fmt.Errorf("task_service: no active approval matrix configured")
	}

	now := time.Now()

	header := &model.ApprovalHeader{
		TaskID:        taskID,
		TotalLevel:    chain[len(chain)-1].RoleLevel,
		CurrentLevel:  chain[0].RoleLevel,
		OverallStatus: constants.ApprovalOverallStatusPending,
		SubmittedBy:   submittedBy,
		SubmittedAt:   now,
		CreatedAt:     now.Add(1 * time.Second),
	}
	if err := w.RepositoryApproval.CreateHeader(ctx, header); err != nil {
		return err
	}

	details := make([]*model.ApprovalDetail, 0, len(chain))
	for i, step := range chain {
		status := constants.ApprovalDetailStatusWaiting
		if i == 0 {
			status = constants.ApprovalDetailStatusPending
		}
		code, err := repository.GenerateSecureApprovalCode()
		if err != nil {
			return err
		}
		details = append(details, &model.ApprovalDetail{
			ApprovalHeaderID:   header.ID,
			Level:              step.RoleLevel,
			ApproverRoleID:     step.RoleID,
			ApproverID:         step.UserID,
			Status:             status,
			SecureApprovalCode: code,
			CreatedAt:          now.Add(1 * time.Second),
		})
	}
	if err := w.RepositoryApproval.CreateDetails(ctx, details); err != nil {
		return err
	}
	w.notifyApprovalPending(ctx, taskID, task.SiteID, submittedBy, details[0], notes)

	action, round := constants.TaskActionSubmit, roundOf(task)
	taskUpdates := map[string]interface{}{
		"TaskStatus": constants.TaskStatusWaiting(chain[0].RoleID),
		"UpdatedBy":  submittedBy,
		"UpdatedAt":  now,
	}
	if task.TaskStatus == constants.TaskStatusRevision {
		action, round = constants.TaskActionResubmit, round+1
		taskUpdates["SubmitCount"] = round
	}
	if err := w.RepositoryTask.UpdateStatus(ctx, taskID, taskUpdates); err != nil {
		return err
	}
	historyNotes := fmt.Sprintf("Approval dikirim ke %s, status PENDING", w.roleNameOf(ctx, chain[0].RoleID))
	if notes != "" {
		historyNotes = fmt.Sprintf("%s (catatan: %s)", historyNotes, notes)
	}
	return w.recordHistory(ctx, taskID, action, submittedBy, historyMeta{role: submitterRole, round: round}, historyNotes)
}

// notifyApprovalPending tells the approver of a level that just became
// PENDING: activates its e-mail link code in Redis and sends (or queues)
// the e-mail. Best effort: failures are logged, never returned.
func (w *taskWorkflow) notifyApprovalPending(ctx context.Context, taskID string, siteID string, submittedBy int64, detail *model.ApprovalDetail, notes string) {
	approver, err := w.RepositoryUser.GetByID(ctx, detail.ApproverID)
	if err != nil || approver == nil || approver.Email == "" {
		w.logError(fmt.Errorf("task_service: skip approval-pending email for task %s (approver %d has no email): %v", taskID, detail.ApproverID, err))
		return
	}

	siteName := siteID
	if site, err := w.RepositorySite.GetByID(ctx, siteID); err == nil && site != nil {
		siteName = site.SiteName1
	}

	if err := w.RepositoryApprovalCode.Set(ctx, detail.SecureApprovalCode, detail.ID); err != nil {
		w.logError(fmt.Errorf("task_service: failed to cache approval code in redis for task %s: %w", taskID, err))
	}

	payload := dto.ApprovalEmailTaskPayload{
		To:           approver.Email,
		ToName:       approver.Username,
		TaskID:       taskID,
		SiteName:     siteName,
		SubmittedBy:  usernameOf(ctx, w.RepositoryUser, submittedBy),
		ApproverRole: w.roleNameOf(ctx, detail.ApproverRoleID),
		Notes:        notes,
		ApprovalCode: detail.SecureApprovalCode,
	}

	if !w.ClientTaskQueue.Enabled() {
		w.sendApprovalPendingEmail(ctx, payload)
		return
	}

	body, err := json.Marshal(payload)
	if err != nil {
		w.logError(fmt.Errorf("task_service: failed to marshal approval-pending payload for task %s: %w", taskID, err))
		return
	}
	if err := w.ClientTaskQueue.Enqueue(ctx, fmt.Sprintf("approval-email-%s", taskID), "/v1/queue/approval-email", body); err != nil {
		w.logError(fmt.Errorf("task_service: failed to enqueue approval-pending email for task %s: %w", taskID, err))
	}
}

// sendApprovalPendingEmail sends at once (queue off, or the queue
// callback).
func (w *taskWorkflow) sendApprovalPendingEmail(ctx context.Context, payload dto.ApprovalEmailTaskPayload) error {
	err := w.ClientMail.SendApprovalPending(ctx, payload.To, payload.ToName, client.ApprovalPendingEmail{
		TaskID:       payload.TaskID,
		SiteName:     payload.SiteName,
		SubmittedBy:  payload.SubmittedBy,
		ApproverRole: payload.ApproverRole,
		Notes:        payload.Notes,
		Code:         payload.ApprovalCode,
	})
	if err != nil {
		w.logError(fmt.Errorf("task_service: failed to send approval-pending email for task %s to %s: %w", payload.TaskID, payload.To, err))
	}
	return err
}

// enqueueAIInsight queues the AI insight of a site (queue off: generates
// at once). actorID is carried so the insight still records who caused it.
func (w *taskWorkflow) enqueueAIInsight(ctx context.Context, siteID string, actorID int64) {
	if !w.ClientTaskQueue.Enabled() {
		if _, err := w.ServicesAIInsight.Generate(ctx, siteID, actorID); err != nil {
			w.logError(fmt.Errorf("task_service: failed to generate AI insight for site %s: %w", siteID, err))
		}
		return
	}

	body, err := json.Marshal(dto.AIInsightTaskPayload{SiteID: siteID, ActorID: actorID})
	if err != nil {
		w.logError(fmt.Errorf("task_service: failed to marshal AI insight payload for site %s: %w", siteID, err))
		return
	}
	if err := w.ClientTaskQueue.Enqueue(ctx, fmt.Sprintf("ai-insight-%s", siteID), "/v1/queue/ai-insight", body); err != nil {
		w.logError(fmt.Errorf("task_service: failed to enqueue AI insight for site %s: %w", siteID, err))
	}
}

// IQueueService runs the background jobs the /v1/queue/* callbacks receive.
type IQueueService interface {
	// SendApprovalEmail is POST /v1/queue/approval-email.
	SendApprovalEmail(ctx context.Context, payload dto.ApprovalEmailTaskPayload) (err error)
}

type QueueService struct {
	opt      opt.Options
	workflow *taskWorkflow
}

func NewQueueService() IQueueService {
	return &QueueService{
		opt:      global.Setup.Common,
		workflow: newTaskWorkflow(),
	}
}

func (s *QueueService) SendApprovalEmail(ctx context.Context, payload dto.ApprovalEmailTaskPayload) (err error) {
	ctx, span := s.opt.Trace.Start(ctx, "QueueService.SendApprovalEmail")
	defer span.End()

	if err = s.workflow.sendApprovalPendingEmail(ctx, payload); err != nil {
		span.RecordError(err)
		return
	}

	return
}
