package handler

import (
	"strconv"

	"site-dev-backend/global"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/middleware"
	"site-dev-backend/internal/app/service"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs"
	"gitlab.com/fami-app-v2/familib/familabs/log"
)

// UserHandler is /v1/users: the admin "Pengguna" screen.
type UserHandler struct {
	Option         global.HandlerOptions
	ServicesAccess service.IAccessService
	ServicesUser   service.IUserService
}

// URLMapping implements router.RouteHandlers
func (h *UserHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesAccess = service.NewAccessService()
	h.ServicesUser = service.NewUserService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *UserHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.GET("", h.Index, mw.Authorized())
	r.POST("", h.Create, mw.Authorized())
	r.PUT("/:id", h.Update, mw.Authorized())
	r.DELETE("/:id", h.Delete, mw.Authorized())
}

// Index is GET /v1/users (admin only).
func (h UserHandler) Index(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	if err = h.ServicesAccess.RequireAdmin(ctx.Request().Context(), userIDFromContext(ctx)); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData, err = h.ServicesUser.Get(ctx.Request().Context())
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Create is POST /v1/users (admin only).
func (h UserHandler) Create(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	if err = h.ServicesAccess.RequireAdmin(ctx.Request().Context(), userIDFromContext(ctx)); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var req dto.UserRequestCreate
	if err = ctx.Bind(&req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData, err = h.ServicesUser.Create(ctx.Request().Context(), req)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Update is PUT /v1/users/:id (admin only).
func (h UserHandler) Update(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	if err = h.ServicesAccess.RequireAdmin(ctx.Request().Context(), userIDFromContext(ctx)); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var id int64
	if id, err = strconv.ParseInt(ctx.GetUriParamString("id"), 10, 64); err != nil {
		err = familabs.ErrorInvalid("id")
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var req dto.UserRequestUpdate
	if err = ctx.Bind(&req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData, err = h.ServicesUser.Update(ctx.Request().Context(), id, req)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Delete is DELETE /v1/users/:id (admin only).
func (h UserHandler) Delete(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	if err = h.ServicesAccess.RequireAdmin(ctx.Request().Context(), userIDFromContext(ctx)); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var id int64
	if id, err = strconv.ParseInt(ctx.GetUriParamString("id"), 10, 64); err != nil {
		err = familabs.ErrorInvalid("id")
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	if err = h.ServicesUser.Delete(ctx.Request().Context(), id); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}
