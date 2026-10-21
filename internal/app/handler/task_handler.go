package handler

import (
	"site-dev-backend/global"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/middleware"
	"site-dev-backend/internal/app/service"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs"
	"gitlab.com/fami-app-v2/familib/familabs/constants"
	"gitlab.com/fami-app-v2/familib/familabs/log"
)

type TaskHandler struct {
	Option       global.HandlerOptions
	ServicesTask service.ITaskService
}

// URLMapping implements router.RouteHandlers
func (h *TaskHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesTask = service.NewTaskService()

	mw := middleware.NewMiddleware()

	r.GET("", h.Index, mw.Authorized())
	r.GET("/:id/history", h.History, mw.Authorized())
}

// Index is GET /v1/task: the caller's task list (site staff see only their
// own tasks), 20 rows per page.
func (h TaskHandler) Index(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	// get params pagination
	var page *familabs.Paginator
	page, err = familabs.NewPaginator(ctx)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	req := dto.TaskRequestGet{
		Page:    page.Page,
		SiteID:  ctx.GetParamString("site_id"),
		Status:  ctx.GetParamString("status"),
		Search:  ctx.GetParamString("q"),
		SortBy:  ctx.GetParamString("sort_by"),
		SortDir: ctx.GetParamString("sort_dir"),
	}

	var tasks []dto.TaskResponseGet
	var total int64
	tasks, total, err = h.ServicesTask.Get(ctx.Request().Context(), req, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.DataList(tasks, total, page)

	return ctx.Serve(err)
}

// History is GET /v1/task/:id/history (id = site_id): the task's history
// rows, oldest first.
func (h TaskHandler) History(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	siteID := ctx.GetUriParamString("id")

	var histories []dto.TaskHistoryResponseGet
	histories, err = h.ServicesTask.GetHistory(ctx.Request().Context(), siteID)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData = histories

	return ctx.Serve(err)
}

// userIDFromContext reads the user ID that middlewarex.Authorization (used
// by Middleware.Authorized) stores in the request context.
func userIDFromContext(ctx *familabs.Context) int64 {
	userID, _ := ctx.Request().Context().Value(constants.KeyUserID).(int64)
	return userID
}
