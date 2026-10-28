package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ITaskService interface {
	// Get lists tasks for GET /v1/task. Site staff (role USER) only see
	// their own tasks; other roles see every task.
	Get(ctx context.Context, req dto.TaskRequestGet, userID int64) (res []dto.TaskResponseGet, total int64, err error)
	// GetHistory returns the history rows of the task tied to siteID, for
	// GET /v1/task/:id/history.
	GetHistory(ctx context.Context, siteID string) (res []dto.TaskHistoryResponseGet, err error)
	// GetDetail is GET /v1/task/:id: id is a task_id ("TS...") or a site_id
	// ("ST..."). Site staff may only open their own sites (403).
	GetDetail(ctx context.Context, id string, userID int64) (res dto.TaskDetailResponseGet, err error)
	// GetAttachments is GET /v1/task/:id/attachment (id = site_id).
	GetAttachments(ctx context.Context, siteID string) (res []dto.TaskAttachmentResponseGet, err error)
	// CountByStatus is GET /v1/task/status/count: site staff count only
	// their own tasks.
	CountByStatus(ctx context.Context, userID int64) (res []dto.TaskStatusCountResponseGet, err error)
	// GetSiteList is GET /v1/task/site-list: every task in
	// constants.SiteListRequestStatuses, for every role.
	GetSiteList(ctx context.Context, req dto.ListRequestGet) (res []dto.TaskResponseGet, total int64, err error)
	// GetPendingApprovals is GET /v1/task/pending-approvals: the levels
	// waiting for this approver.
	GetPendingApprovals(ctx context.Context, req dto.ListRequestGet, approverID int64) (res []dto.PendingApprovalResponseGet, total int64, err error)
	// GetApprovalHistory is GET /v1/task/approval-history: the levels this
	// approver already decided.
	GetApprovalHistory(ctx context.Context, req dto.ListRequestGet, approverID int64) (res []dto.ApprovalHistoryResponseGet, total int64, err error)
	// GetDashboardHistory is GET /v1/dashboard/task/history: history rows of
	// every task, no role filter.
	GetDashboardHistory(ctx context.Context, req dto.ListRequestGet) (res []dto.DashboardTaskHistoryResponseGet, total int64, err error)
	// GetDashboardStatusCount is GET /v1/dashboard/task/status/count: every
	// task, no role filter.
	GetDashboardStatusCount(ctx context.Context) (res []dto.TaskStatusCountResponseGet, err error)
	// Approve is POST /v1/task/:id/approve (id = task_id): the caller's
	// decision on the level waiting for them; APPROVE moves to the next
	// level or completes the task.
	Approve(ctx context.Context, taskID string, actorID int64, req dto.TaskActionRequest) (err error)
	// Cancel is POST /v1/task/:id/cancel (Wes, 26 Oct, "Batalkan
	// pengajuan"): the creator closes a request returned to them (REVISION)
	// as CANCELLED, optional notes (403 not the creator, 409 other status,
	// 404 unknown).
	Cancel(ctx context.Context, id string, actorID int64, req dto.TaskCancelRequest) (err error)
	// GetNearbyRequests is GET /v1/task/nearby (local §8): active requests
	// of all staff around a point, nearest first.
	GetNearbyRequests(ctx context.Context, req dto.NearbyRequestGet) (res []dto.NearbyRequestResponseGet, err error)
}

type TaskService struct {
	opt                   opt.Options
	RepositoryTask        repository.ITaskRepository
	RepositoryTaskHistory repository.ITaskHistoryRepository
	RepositorySite        repository.ISiteRepository
	RepositoryUser        repository.IUserRepository
	RepositorySiteContext repository.ISiteContextRepository
	RepositoryApproval    repository.IApprovalRepository
	RepositoryTransaction repository.ITransactionRepository
	workflow              *taskWorkflow
}

func NewTaskService() ITaskService {
	return newTaskService()
}

// newTaskService is NewTaskService as the concrete type (SiteService reuses
// its task detail).
func newTaskService() *TaskService {
	return &TaskService{
		opt:                   global.Setup.Common,
		RepositoryTask:        repository.NewTaskRepository(),
		RepositoryTaskHistory: repository.NewTaskHistoryRepository(),
		RepositorySite:        repository.NewSiteRepository(),
		RepositoryUser:        repository.NewUserRepository(),
		RepositorySiteContext: repository.NewSiteContextRepository(),
		RepositoryApproval:    repository.NewApprovalRepository(),
		RepositoryTransaction: repository.NewTransactionRepository(),
		workflow:              newTaskWorkflow(),
	}
}

func (s *TaskService) Get(ctx context.Context, req dto.TaskRequestGet, userID int64) (res []dto.TaskResponseGet, total int64, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "TaskService.Get")
	defer span.End()

	var user *model.User
	user, err = getUser(ctx, s.RepositoryUser, userID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	var siteIDs []string
	if req.SiteID != "" {
		siteIDs = []string{req.SiteID}
	}
	var createdBy int64
	if user.RoleID == constants.RoleUser {
		createdBy = userID
	}
	if req.Search != "" {
		siteIDs, err = s.RepositorySite.GetIDsBySearch(ctx, req.Search)
		if err != nil {
			span.RecordError(err)
			s.opt.Logger.AddMessage(log.ErrorLevel, err)
			return
		}
		if len(siteIDs) == 0 {
			res = []dto.TaskResponseGet{}
			return
		}
	}

	offset := pageOffset(req.Page)
	orderBy := taskOrderBy(req.SortBy, req.SortDir)

	var tasks []*model.Task
	if req.Status == constants.TaskStatusHoldGroup {
		// status=HOLD = both HOLD_* (November scope, API-CHANGES §13.3)
		tasks, total, err = s.RepositoryTask.GetByStatuses(ctx, offset, dto.TaskPageSize, orderBy, constants.HoldStatuses, siteIDs, createdBy)
	} else if req.Status == constants.TaskStatusWaitingGroup {
		// status=WAITING = "Menunggu": both waiting levels (Wes, 28 Oct, simple app)
		tasks, total, err = s.RepositoryTask.GetByStatuses(ctx, offset, dto.TaskPageSize, orderBy, constants.WaitingStatuses, siteIDs, createdBy)
	} else if req.Status == constants.TaskStatusRejectedGroup {
		// status=REJECTED_ANY = "Ditolak": REVISION + REJECTED (Wes, 26 Oct)
		tasks, total, err = s.RepositoryTask.GetByStatuses(ctx, offset, dto.TaskPageSize, orderBy, constants.RejectedStatuses, siteIDs, createdBy)
	} else if req.Status == "" && len(req.StatusNot) > 0 {
		// status_not (November scope, open: review §5 #7)
		tasks, total, err = s.RepositoryTask.GetNotInStatuses(ctx, offset, dto.TaskPageSize, orderBy, req.StatusNot, siteIDs, createdBy)
	} else {
		tasks, total, err = s.RepositoryTask.Get(ctx, offset, dto.TaskPageSize, orderBy, siteIDs, req.Status, createdBy)
	}
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}
	span.AddEvent("tasks are loaded", trace.WithAttributes(attribute.Int64("total", total)))

	res, err = s.withSites(ctx, tasks)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	return
}

// withSites adds the site name and address lines to every task row (one
// query for all sites).
func (s *TaskService) withSites(ctx context.Context, tasks []*model.Task) (res []dto.TaskResponseGet, err error) {
	taskSiteIDs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskSiteIDs = append(taskSiteIDs, task.SiteID)
	}
	var sites []*model.Site
	sites, err = s.RepositorySite.GetByIDs(ctx, taskSiteIDs)
	if err != nil {
		return
	}
	siteByID := make(map[string]*model.Site, len(sites))
	for _, site := range sites {
		siteByID[site.SiteID] = site
	}

	var rejectedBy map[string]string
	if rejectedBy, err = s.rejectedByRoles(ctx, tasks); err != nil {
		return
	}

	res = make([]dto.TaskResponseGet, 0, len(tasks))
	for _, task := range tasks {
		row := toTaskResponse(task, siteByID[task.SiteID])
		row.RejectedByRole = rejectedBy[task.TaskID]
		res = append(res, row)
	}
	return
}

// returnedStatuses are the task statuses that have a "returned by" role,
// with the level status that caused each.
var returnedStatuses = map[string]string{
	constants.TaskStatusRevision: constants.ApprovalDetailStatusRevision,
	constants.TaskStatusRejected: constants.ApprovalDetailStatusRejected,
}

// rejectedByRoles maps task ID -> role of the level that returned (REVISION)
// or rejected it, for the tasks now in such a status (November scope,
// prototype rejected_by_role). One query, only when such a task is listed.
func (s *TaskService) rejectedByRoles(ctx context.Context, tasks []*model.Task) (res map[string]string, err error) {
	var taskIDs []string
	for _, t := range tasks {
		if _, ok := returnedStatuses[t.TaskStatus]; ok {
			taskIDs = append(taskIDs, t.TaskID)
		}
	}
	if len(taskIDs) == 0 {
		return
	}

	var rows []model.TaskDecisionRole
	rows, err = s.RepositoryApproval.GetDecisionRolesByTaskIDs(ctx, taskIDs, []string{constants.ApprovalDetailStatusRevision, constants.ApprovalDetailStatusRejected})
	if err != nil {
		return
	}
	res = make(map[string]string, len(taskIDs))
	for _, row := range rows { // oldest first: the newest decision wins
		res[row.TaskID] = row.ApproverRoleID
	}
	return
}

func (s *TaskService) GetHistory(ctx context.Context, siteID string) (res []dto.TaskHistoryResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "TaskService.GetHistory")
	defer span.End()

	var task *model.Task
	task, err = s.RepositoryTask.GetBySiteID(ctx, siteID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	var histories []*model.TaskHistory
	histories, err = s.RepositoryTaskHistory.GetByTaskID(ctx, task.TaskID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res = make([]dto.TaskHistoryResponseGet, 0, len(histories))
	for _, history := range histories {
		res = append(res, toTaskHistoryResponse(history))
	}

	return
}

func (s *TaskService) GetDetail(ctx context.Context, id string, userID int64) (res dto.TaskDetailResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "TaskService.GetDetail")
	defer span.End()

	var user *model.User
	user, err = getUser(ctx, s.RepositoryUser, userID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res, err = s.detail(ctx, id)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	if user.RoleID == constants.RoleUser && res.Site.CreatedBy != userID {
		err = echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("Anda tidak punya akses ke task %s", res.Task.TaskID))
		span.RecordError(err)
		res = dto.TaskDetailResponseGet{}
		return
	}

	return
}

// detail builds the task detail without the access check (also used as the
// "task" part of the save response).
func (s *TaskService) detail(ctx context.Context, id string) (res dto.TaskDetailResponseGet, err error) {
	var task *model.Task
	switch {
	case strings.HasPrefix(id, constants.TaskIDPrefix):
		task, err = s.RepositoryTask.GetByID(ctx, id)
	case constants.IsSiteID(id):
		task, err = s.RepositoryTask.GetBySiteID(ctx, id)
	default:
		err = fmt.Errorf("id %q tidak dikenali (harus task_id atau site_id)", id)
	}
	if err != nil {
		return
	}

	var site *model.Site
	site, err = s.RepositorySite.GetByID(ctx, task.SiteID)
	if err != nil {
		return
	}

	res.Site = toSiteResponse(site)
	res.Site.CreatedByName = usernameOf(ctx, s.RepositoryUser, site.CreatedBy)
	if site.UpdatedBy != nil {
		res.Site.UpdatedByName = usernameOf(ctx, s.RepositoryUser, *site.UpdatedBy)
	}
	res.Task = dto.TaskDetailTaskSection{
		TaskID:      task.TaskID,
		TaskStatus:  task.TaskStatus,
		SubmitCount: task.SubmitCount,
	}

	// the on-site survey is optional: never filled in = null, not an error
	siteContext, ctxErr := s.RepositorySiteContext.GetBySiteID(ctx, task.SiteID)
	if ctxErr == nil {
		var competitors []*model.SiteCompetitor
		competitors, err = s.RepositorySiteContext.GetCompetitorsBySiteID(ctx, task.SiteID)
		if err != nil {
			return
		}
		var notes []*model.SiteCustomNote
		notes, err = s.RepositorySiteContext.GetCustomNotesBySiteID(ctx, task.SiteID)
		if err != nil {
			return
		}
		contextRes := toSiteContextResponse(siteContext, competitors, notes)
		res.Context = &contextRes
	} else if err = ignoreNotFound(ctxErr); err != nil {
		return
	}

	var histories []*model.TaskHistory
	histories, err = s.RepositoryTaskHistory.GetByTaskID(ctx, task.TaskID)
	if err != nil {
		return
	}
	for _, history := range histories {
		res.Task.History = append(res.Task.History, toTaskHistoryResponse(history))
	}

	// never submitted = no approval round yet, approval stays null
	header, headerErr := s.RepositoryApproval.GetHeaderByTaskID(ctx, task.TaskID)
	if headerErr == nil {
		var details []*model.ApprovalDetail
		details, err = s.RepositoryApproval.GetDetailsByHeaderID(ctx, header.ID)
		if err != nil {
			return
		}
		levelStatus, returned := returnedStatuses[task.TaskStatus]
		for _, d := range details {
			if returned && d.Status == levelStatus {
				res.Task.RejectedByRole = d.ApproverRoleID // the newest round holds the decision
			}
			res.Task.Approval = append(res.Task.Approval, dto.TaskDetailApprovalItem{
				ID:             d.ID,
				Level:          d.Level,
				ApproverRoleID: d.ApproverRoleID,
				ApproverID:     d.ApproverID,
				ApproverName:   usernameOf(ctx, s.RepositoryUser, d.ApproverID),
				Status:         d.Status,
				Notes:          d.Notes,
				CreatedAt:      d.CreatedAt,
				ActedAt:        d.ActedAt,
			})
		}
	} else if err = ignoreNotFound(headerErr); err != nil {
		return
	}

	var attachments []*model.SiteAttachment
	attachments, err = s.RepositorySite.GetAttachmentsBySiteID(ctx, task.SiteID)
	if err != nil {
		return
	}
	res.Attachment = toAttachmentResponses(attachments, site.CreatedBy)

	return
}

func (s *TaskService) GetAttachments(ctx context.Context, siteID string) (res []dto.TaskAttachmentResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "TaskService.GetAttachments")
	defer span.End()

	var site *model.Site
	site, err = s.RepositorySite.GetByID(ctx, siteID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	var attachments []*model.SiteAttachment
	attachments, err = s.RepositorySite.GetAttachmentsBySiteID(ctx, siteID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	// this list is [] when empty (the detail's "attachment" is null)
	res = make([]dto.TaskAttachmentResponseGet, 0, len(attachments))
	res = append(res, toAttachmentResponses(attachments, site.CreatedBy)...)

	return
}

func (s *TaskService) CountByStatus(ctx context.Context, userID int64) (res []dto.TaskStatusCountResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "TaskService.CountByStatus")
	defer span.End()

	var user *model.User
	user, err = getUser(ctx, s.RepositoryUser, userID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	var createdBy int64
	if user.RoleID == constants.RoleUser {
		createdBy = userID
	}

	res, err = s.countByStatus(ctx, createdBy)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	return
}

func (s *TaskService) countByStatus(ctx context.Context, createdBy int64) (res []dto.TaskStatusCountResponseGet, err error) {
	var counts []model.TaskStatusCount
	counts, err = s.RepositoryTask.CountByStatus(ctx, createdBy)
	if err != nil {
		return
	}

	res = make([]dto.TaskStatusCountResponseGet, 0, len(counts))
	for _, c := range counts {
		res = append(res, dto.TaskStatusCountResponseGet{TaskStatus: c.TaskStatus, Count: c.Count})
	}
	return
}

func (s *TaskService) GetSiteList(ctx context.Context, req dto.ListRequestGet) (res []dto.TaskResponseGet, total int64, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "TaskService.GetSiteList")
	defer span.End()

	var siteIDs []string
	if req.Search != "" {
		siteIDs, err = s.RepositorySite.GetIDsBySearch(ctx, req.Search)
		if err != nil {
			span.RecordError(err)
			s.opt.Logger.AddMessage(log.ErrorLevel, err)
			return
		}
		if len(siteIDs) == 0 {
			res = []dto.TaskResponseGet{}
			return
		}
	}

	var tasks []*model.Task
	tasks, total, err = s.RepositoryTask.GetByStatuses(ctx, pageOffset(req.Page), dto.ListPageSize, taskOrderBy(req.SortBy, req.SortDir), constants.SiteListRequestStatuses, siteIDs, 0)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res, err = s.withSites(ctx, tasks)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	return
}

// approvalRow is the round, task, site and creator name of one ApprovalDET
// row.
type approvalRow struct {
	header        *model.ApprovalHeader
	task          *model.Task
	site          *model.Site
	createdByName string
}

// approvalRowOf loads the round, task, site and creator name of one level
// (one query each, as in site-dev-api).
func (s *TaskService) approvalRowOf(ctx context.Context, d *model.ApprovalDetail) (row approvalRow, err error) {
	row.header, err = s.RepositoryApproval.GetHeaderByID(ctx, d.ApprovalHeaderID)
	if err != nil {
		return
	}
	row.task, err = s.RepositoryTask.GetByID(ctx, row.header.TaskID)
	if err != nil {
		return
	}
	row.site, err = s.RepositorySite.GetByID(ctx, row.task.SiteID)
	if err != nil {
		return
	}
	if creator, userErr := s.RepositoryUser.GetByID(ctx, row.site.CreatedBy); userErr == nil {
		row.createdByName = creator.Username
	}
	return
}

func (s *TaskService) GetPendingApprovals(ctx context.Context, req dto.ListRequestGet, approverID int64) (res []dto.PendingApprovalResponseGet, total int64, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "TaskService.GetPendingApprovals")
	defer span.End()

	orderBy := "ApprovalDET." + orderClause(req.SortBy, req.SortDir, "CreatedAt", map[string]bool{"CreatedAt": true, "Level": true})

	var details []*model.ApprovalDetail
	details, total, err = s.RepositoryApproval.GetPendingByApprover(ctx, approverID, req.Search, orderBy, dto.ListPageSize, pageOffset(req.Page))
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res = make([]dto.PendingApprovalResponseGet, 0, len(details))
	for _, d := range details {
		var row approvalRow
		row, err = s.approvalRowOf(ctx, d)
		if err != nil {
			span.RecordError(err)
			s.opt.Logger.AddMessage(log.ErrorLevel, err)
			res, total = nil, 0
			return
		}
		res = append(res, dto.PendingApprovalResponseGet{
			ApprovalDetailID: d.ID,
			TaskID:           row.task.TaskID,
			SiteID:           row.site.SiteID,
			SiteName1:        row.site.SiteName1,
			SiteAddress1:     row.site.SiteAddress1,
			SiteAddress2:     row.site.SiteAddress2,
			SiteAddress3:     row.site.SiteAddress3,
			TaskStatus:       row.task.TaskStatus,
			CreatedBy:        row.site.CreatedBy,
			CreatedByName:    row.createdByName,
			Level:            d.Level,
			SubmittedAt:      row.header.SubmittedAt,
			CreatedAt:        row.task.CreatedAt,
			SubmitCount:      row.task.SubmitCount,
		})
	}

	return
}

func (s *TaskService) GetApprovalHistory(ctx context.Context, req dto.ListRequestGet, approverID int64) (res []dto.ApprovalHistoryResponseGet, total int64, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "TaskService.GetApprovalHistory")
	defer span.End()

	orderBy := "ApprovalDET." + orderClause(req.SortBy, req.SortDir, "ActedAt", map[string]bool{"ActedAt": true, "CreatedAt": true, "Level": true})

	var details []*model.ApprovalDetail
	details, total, err = s.RepositoryApproval.GetActedByApprover(ctx, approverID, req.Search, orderBy, dto.ListPageSize, pageOffset(req.Page))
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res = make([]dto.ApprovalHistoryResponseGet, 0, len(details))
	var tasks []*model.Task
	for _, d := range details {
		var row approvalRow
		row, err = s.approvalRowOf(ctx, d)
		if err != nil {
			span.RecordError(err)
			s.opt.Logger.AddMessage(log.ErrorLevel, err)
			res, total = nil, 0
			return
		}
		res = append(res, dto.ApprovalHistoryResponseGet{
			ApprovalDetailID: d.ID,
			TaskID:           row.task.TaskID,
			SiteID:           row.site.SiteID,
			SiteName1:        row.site.SiteName1,
			SiteAddress1:     row.site.SiteAddress1,
			SiteAddress2:     row.site.SiteAddress2,
			SiteAddress3:     row.site.SiteAddress3,
			TaskStatus:       row.task.TaskStatus,
			CreatedBy:        row.site.CreatedBy,
			CreatedByName:    row.createdByName,
			Level:            d.Level,
			Status:           d.Status,
			Notes:            d.Notes,
			ActedAt:          d.ActedAt,
			CreatedAt:        row.task.CreatedAt,
		})
		tasks = append(tasks, row.task)
	}

	var rejectedBy map[string]string
	if rejectedBy, err = s.rejectedByRoles(ctx, tasks); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		res, total = nil, 0
		return
	}
	for i := range res {
		res[i].RejectedByRole = rejectedBy[res[i].TaskID]
	}

	return
}

func (s *TaskService) GetDashboardHistory(ctx context.Context, req dto.ListRequestGet) (res []dto.DashboardTaskHistoryResponseGet, total int64, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "TaskService.GetDashboardHistory")
	defer span.End()

	orderBy := orderClause(req.SortBy, req.SortDir, "CreatedAt", map[string]bool{"CreatedAt": true})

	var histories []*model.TaskHistory
	histories, total, err = s.RepositoryTaskHistory.Get(ctx, pageOffset(req.Page), dto.ListPageSize, orderBy)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	taskIDs := make([]string, 0, len(histories))
	seen := map[string]bool{}
	for _, h := range histories {
		if !seen[h.TaskID] {
			seen[h.TaskID] = true
			taskIDs = append(taskIDs, h.TaskID)
		}
	}

	taskByID := map[string]*model.Task{}
	siteByID := map[string]*model.Site{}
	if len(taskIDs) > 0 {
		var tasks []*model.Task
		tasks, err = s.RepositoryTask.GetByIDs(ctx, taskIDs)
		if err != nil {
			span.RecordError(err)
			s.opt.Logger.AddMessage(log.ErrorLevel, err)
			res, total = nil, 0
			return
		}
		siteIDs := make([]string, 0, len(tasks))
		for _, t := range tasks {
			taskByID[t.TaskID] = t
			siteIDs = append(siteIDs, t.SiteID)
		}

		var sites []*model.Site
		sites, err = s.RepositorySite.GetByIDs(ctx, siteIDs)
		if err != nil {
			span.RecordError(err)
			s.opt.Logger.AddMessage(log.ErrorLevel, err)
			res, total = nil, 0
			return
		}
		for _, site := range sites {
			siteByID[site.SiteID] = site
		}
	}

	res = make([]dto.DashboardTaskHistoryResponseGet, 0, len(histories))
	for _, h := range histories {
		item := dto.DashboardTaskHistoryResponseGet{
			ID:        h.ID,
			TaskID:    h.TaskID,
			Action:    h.Action,
			ActorID:   h.ActorID,
			ActorName: usernameOf(ctx, s.RepositoryUser, h.ActorID),
			Notes:     h.Notes,
			CreatedAt: h.CreatedAt,
		}
		if task := taskByID[h.TaskID]; task != nil {
			item.SiteID = task.SiteID
			item.TaskStatus = task.TaskStatus
			if site := siteByID[task.SiteID]; site != nil {
				item.SiteName1 = site.SiteName1
			}
		}
		res = append(res, item)
	}

	return
}

func (s *TaskService) GetDashboardStatusCount(ctx context.Context) (res []dto.TaskStatusCountResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "TaskService.GetDashboardStatusCount")
	defer span.End()

	res, err = s.countByStatus(ctx, 0)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	return
}

// taskOrderBy builds the ORDER BY of the task lists: default "CreatedAt
// ASC"; sort_by may be CreatedAt or TaskStatus; sort_dir desc/-1 or asc/1.
func taskOrderBy(sortBy string, sortDir string) string {
	return orderClause(sortBy, sortDir, "CreatedAt", map[string]bool{"CreatedAt": true, "TaskStatus": true})
}
