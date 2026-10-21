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

// SettingsHandler is /v1/settings: daily limits and quota reset (admin only).
type SettingsHandler struct {
	Option           global.HandlerOptions
	ServicesAccess   service.IAccessService
	ServicesSettings service.ISettingsService
}

// URLMapping implements router.RouteHandlers
func (h *SettingsHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesAccess = service.NewAccessService()
	h.ServicesSettings = service.NewSettingsService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *SettingsHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.GET("/limits", h.Limits, mw.Authorized())
	r.PUT("/limits", h.UpdateLimits, mw.Authorized())
	r.POST("/quota-flush", h.QuotaFlush, mw.Authorized())
}

// Limits is GET /v1/settings/limits.
func (h SettingsHandler) Limits(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	if err = h.ServicesAccess.RequireAdmin(ctx.Request().Context(), userIDFromContext(ctx)); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData, err = h.ServicesSettings.GetLimits(ctx.Request().Context())
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// UpdateLimits is PUT /v1/settings/limits.
func (h SettingsHandler) UpdateLimits(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	if err = h.ServicesAccess.RequireAdmin(ctx.Request().Context(), userIDFromContext(ctx)); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var req dto.LimitsRequestUpdate
	if err = ctx.Bind(&req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData, err = h.ServicesSettings.UpdateLimits(ctx.Request().Context(), req, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// QuotaFlush is POST /v1/settings/quota-flush ("Reset Limit Pengguna").
func (h SettingsHandler) QuotaFlush(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	if err = h.ServicesAccess.RequireAdmin(ctx.Request().Context(), userIDFromContext(ctx)); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var req dto.QuotaFlushRequest
	if err = ctx.Bind(&req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData, err = h.ServicesSettings.FlushUserQuota(ctx.Request().Context(), req)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}
