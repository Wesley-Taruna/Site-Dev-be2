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
	Option              global.HandlerOptions
	ServicesTask        service.ITaskService
	ServicesSiteContext service.ISiteContextService
}

// URLMapping implements router.RouteHandlers
func (h *TaskHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesTask = service.NewTaskService()
	h.ServicesSiteContext = service.NewSiteContextService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *TaskHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.GET("", h.Index, mw.Authorized())
	r.GET("/site-list", h.SiteList, mw.Authorized())
	r.GET("/pending-approvals", h.PendingApprovals, mw.Authorized())
	r.GET("/approval-history", h.ApprovalHistory, mw.Authorized())
	r.GET("/status/count", h.StatusCount, mw.Authorized())
	r.GET("/:id", h.Detail, mw.Authorized())
	r.GET("/:id/history", h.History, mw.Authorized())
	r.GET("/:id/attachment", h.Attachment, mw.Authorized())
	r.PUT("/context/:site_id", h.UpdateContext, mw.Authorized())
}

// UpdateContext is PUT /v1/task/context/:site_id: replaces the on-site
// survey (creator only, DRAFT/REVISION only).
func (h TaskHandler) UpdateContext(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.SiteContextRequestUpsert
	if err = ctx.Bind(&req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData, err = h.ServicesSiteContext.Update(ctx.Request().Context(), ctx.GetUriParamString("site_id"), req, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
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

// Detail is GET /v1/task/:id (id = task_id or site_id): site, on-site
// context, approval chain, history and attachments in one answer.
func (h TaskHandler) Detail(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesTask.GetDetail(ctx.Request().Context(), ctx.GetUriParamString("id"), userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Attachment is GET /v1/task/:id/attachment (id = site_id): the site's
// attachments only.
func (h TaskHandler) Attachment(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesTask.GetAttachments(ctx.Request().Context(), ctx.GetUriParamString("id"))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// StatusCount is GET /v1/task/status/count: tasks per status (site staff:
// only their own).
func (h TaskHandler) StatusCount(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesTask.CountByStatus(ctx.Request().Context(), userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// SiteList is GET /v1/task/site-list: the "Site Request" table, every
// role sees every site.
func (h TaskHandler) SiteList(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.ListRequestGet
	var page *familabs.Paginator
	req, page, err = listRequest(ctx)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var tasks []dto.TaskResponseGet
	var total int64
	tasks, total, err = h.ServicesTask.GetSiteList(ctx.Request().Context(), req)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.DataList(tasks, total, page)

	return ctx.Serve(err)
}

// PendingApprovals is GET /v1/task/pending-approvals: what waits for the
// caller's decision.
func (h TaskHandler) PendingApprovals(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.ListRequestGet
	var page *familabs.Paginator
	req, page, err = listRequest(ctx)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var rows []dto.PendingApprovalResponseGet
	var total int64
	rows, total, err = h.ServicesTask.GetPendingApprovals(ctx.Request().Context(), req, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.DataList(rows, total, page)

	return ctx.Serve(err)
}

// ApprovalHistory is GET /v1/task/approval-history: what the caller
// already decided.
func (h TaskHandler) ApprovalHistory(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.ListRequestGet
	var page *familabs.Paginator
	req, page, err = listRequest(ctx)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var rows []dto.ApprovalHistoryResponseGet
	var total int64
	rows, total, err = h.ServicesTask.GetApprovalHistory(ctx.Request().Context(), req, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.DataList(rows, total, page)

	return ctx.Serve(err)
}

// listRequest reads page, sort_by, sort_dir and q of a paged list.
func listRequest(ctx *familabs.Context) (req dto.ListRequestGet, page *familabs.Paginator, err error) {
	page, err = familabs.NewPaginator(ctx)
	if err != nil {
		return
	}

	req = dto.ListRequestGet{
		Page:    page.Page,
		SortBy:  ctx.GetParamString("sort_by"),
		SortDir: ctx.GetParamString("sort_dir"),
		Search:  ctx.GetParamString("q"),
	}
	return
}

// userIDFromContext reads the user ID that middlewarex.Authorization (used
// by Middleware.Authorized) stores in the request context.
func userIDFromContext(ctx *familabs.Context) int64 {
	userID, _ := ctx.Request().Context().Value(constants.KeyUserID).(int64)
	return userID
}
