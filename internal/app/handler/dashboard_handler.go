package handler

import (
	"site-dev-backend/global"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/middleware"
	"site-dev-backend/internal/app/service"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs"
	"gitlab.com/fami-app-v2/familib/familabs/log"
)

// DashboardHandler is /v1/dashboard/task: views over every task, without
// the per-user filter of TaskHandler.
type DashboardHandler struct {
	Option       global.HandlerOptions
	ServicesTask service.ITaskService
}

// URLMapping implements router.RouteHandlers
func (h *DashboardHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesTask = service.NewTaskService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *DashboardHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.GET("/history", h.History, mw.Authorized())
	r.GET("/status/count", h.StatusCount, mw.Authorized())
}

// History is GET /v1/dashboard/task/history: history rows of every task.
func (h DashboardHandler) History(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.ListRequestGet
	var page *familabs.Paginator
	req, page, err = listRequest(ctx)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var rows []dto.DashboardTaskHistoryResponseGet
	var total int64
	rows, total, err = h.ServicesTask.GetDashboardHistory(ctx.Request().Context(), req)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.DataList(rows, total, page)

	return ctx.Serve(err)
}

// StatusCount is GET /v1/dashboard/task/status/count: every task per status.
func (h DashboardHandler) StatusCount(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesTask.GetDashboardStatusCount(ctx.Request().Context())
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}
