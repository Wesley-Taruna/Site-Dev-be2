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

// ActivityHandler is /v1/activity: the site staff's Activity Log
// (November scope, API-CHANGES §13.4).
type ActivityHandler struct {
	Option           global.HandlerOptions
	ServicesActivity service.IActivityService
}

// URLMapping implements router.RouteHandlers
func (h *ActivityHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesActivity = service.NewActivityService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *ActivityHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.GET("", h.Index, mw.Authorized())
	r.GET("/unread-count", h.UnreadCount, mw.Authorized())
	r.POST("/seen", h.Seen, mw.Authorized())
}

// Index is GET /v1/activity?filter=all|action|mine&page=&limit=.
func (h ActivityHandler) Index(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var page *familabs.Paginator
	page, err = newListPage(ctx)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	req := dto.ActivityRequestGet{Filter: ctx.GetParamString("filter"), Offset: page.Offset, Limit: page.Limit}

	var rows []dto.ActivityResponseGet
	var total int64
	rows, total, err = h.ServicesActivity.Get(ctx.Request().Context(), req, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.DataList(rows, total, page)

	return ctx.Serve(err)
}

// UnreadCount is GET /v1/activity/unread-count: {count}.
func (h ActivityHandler) UnreadCount(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesActivity.UnreadCount(ctx.Request().Context(), userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Seen is POST /v1/activity/seen: "seen up to now".
func (h ActivityHandler) Seen(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	if err = h.ServicesActivity.MarkSeen(ctx.Request().Context(), userIDFromContext(ctx)); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}
