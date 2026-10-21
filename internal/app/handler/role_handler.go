package handler

import (
	"site-dev-backend/global"
	"site-dev-backend/internal/app/middleware"
	"site-dev-backend/internal/app/service"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs"
	"gitlab.com/fami-app-v2/familib/familabs/log"
)

// RoleHandler is /v1/roles: the role list for the admin dropdowns.
type RoleHandler struct {
	Option         global.HandlerOptions
	ServicesAccess service.IAccessService
	ServicesRole   service.IRoleService
}

// URLMapping implements router.RouteHandlers
func (h *RoleHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesAccess = service.NewAccessService()
	h.ServicesRole = service.NewRoleService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *RoleHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.GET("", h.Index, mw.Authorized())
}

// Index is GET /v1/roles (admin only).
func (h RoleHandler) Index(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	if err = h.ServicesAccess.RequireAdmin(ctx.Request().Context(), userIDFromContext(ctx)); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData, err = h.ServicesRole.Get(ctx.Request().Context())
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}
