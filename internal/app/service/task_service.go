package service

import (
	"context"
	"errors"
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
	"gorm.io/gorm"
)

type ITaskService interface {
	// Get lists tasks for GET /v1/task. Site staff (role USER) only see
	// their own tasks; other roles see every task.
	Get(ctx context.Context, req dto.TaskRequestGet, userID int64) (res []dto.TaskResponseGet, total int64, err error)
	// GetHistory returns the history rows of the task tied to siteID, for
	// GET /v1/task/:id/history.
	GetHistory(ctx context.Context, siteID string) (res []dto.TaskHistoryResponseGet, err error)
}

type TaskService struct {
	opt                   opt.Options
	RepositoryTask        repository.ITaskRepository
	RepositoryTaskHistory repository.ITaskHistoryRepository
	RepositorySite        repository.ISiteRepository
	RepositoryUser        repository.IUserRepository
}

func NewTaskService() ITaskService {
	return &TaskService{
		opt:                   global.Setup.Common,
		RepositoryTask:        repository.NewTaskRepository(),
		RepositoryTaskHistory: repository.NewTaskHistoryRepository(),
		RepositorySite:        repository.NewSiteRepository(),
		RepositoryUser:        repository.NewUserRepository(),
	}
}

func (s *TaskService) Get(ctx context.Context, req dto.TaskRequestGet, userID int64) (res []dto.TaskResponseGet, total int64, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "TaskService.Get")
	defer span.End()

	// the caller's role comes from Users.RoleID, not from the token
	var user *model.User
	user, err = s.RepositoryUser.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = echo.NewHTTPError(http.StatusUnauthorized, "user not found")
		} else {
			err = echo.NewHTTPError(http.StatusInternalServerError, "failed to verify user")
		}
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

	page := req.Page
	if page == 0 {
		page = 1
	}
	offset := (page - 1) * dto.TaskPageSize
	orderBy := taskOrderBy(req.SortBy, req.SortDir)

	var tasks []*model.Task
	tasks, total, err = s.RepositoryTask.Get(ctx, offset, dto.TaskPageSize, orderBy, siteIDs, req.Status, createdBy)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}
	span.AddEvent("tasks are loaded", trace.WithAttributes(attribute.Int64("total", total)))

	// add the site name and address lines to every row
	taskSiteIDs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskSiteIDs = append(taskSiteIDs, task.SiteID)
	}
	var sites []*model.Site
	sites, err = s.RepositorySite.GetByIDs(ctx, taskSiteIDs)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}
	siteByID := make(map[string]*model.Site, len(sites))
	for _, site := range sites {
		siteByID[site.SiteID] = site
	}

	res = make([]dto.TaskResponseGet, 0, len(tasks))
	for _, task := range tasks {
		row := dto.TaskResponseGet{
			TaskID:     task.TaskID,
			SiteID:     task.SiteID,
			TaskStatus: task.TaskStatus,
			CreatedBy:  task.CreatedBy,
			UpdatedBy:  task.UpdatedBy,
			CreatedAt:  task.CreatedAt,
			UpdatedAt:  task.UpdatedAt,
		}
		if site, ok := siteByID[task.SiteID]; ok && site != nil {
			row.SiteName1 = site.SiteName1
			row.SiteAddress1 = site.SiteAddress1
			row.SiteAddress2 = site.SiteAddress2
			row.SiteAddress3 = site.SiteAddress3
		}
		res = append(res, row)
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
		res = append(res, dto.TaskHistoryResponseGet{
			ID:        history.ID,
			Action:    history.Action,
			ActorID:   history.ActorID,
			Notes:     history.Notes,
			CreatedAt: history.CreatedAt,
		})
	}

	return
}

// taskOrderBy builds the ORDER BY of the task list: default "CreatedAt ASC";
// sort_by may be CreatedAt or TaskStatus; sort_dir desc/-1 or asc/1.
func taskOrderBy(sortBy string, sortDir string) string {
	allowed := map[string]bool{"CreatedAt": true, "TaskStatus": true}

	column := "CreatedAt"
	if sortBy != "" && allowed[sortBy] {
		column = sortBy
	}

	direction := "ASC"
	switch strings.ToLower(sortDir) {
	case "desc", "-1":
		direction = "DESC"
	case "asc", "1":
		direction = "ASC"
	}

	return fmt.Sprintf("%s %s", column, direction)
}
