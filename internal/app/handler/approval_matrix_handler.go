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

// ApprovalMatrixHandler is /v1/approval-matrix ("Matriks Persetujuan").
// site-dev-api has no admin check here; kept that way.
type ApprovalMatrixHandler struct {
	Option                 global.HandlerOptions
	ServicesApprovalMatrix service.IApprovalMatrixService
}

// URLMapping implements router.RouteHandlers
func (h *ApprovalMatrixHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesApprovalMatrix = service.NewApprovalMatrixService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *ApprovalMatrixHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.POST("", h.Create, mw.Authorized())
	r.GET("", h.Index, mw.Authorized())
	r.PUT("/:id", h.Update, mw.Authorized())
}

// Index is GET /v1/approval-matrix.
func (h ApprovalMatrixHandler) Index(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesApprovalMatrix.Get(ctx.Request().Context())
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Create is POST /v1/approval-matrix.
func (h ApprovalMatrixHandler) Create(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.ApprovalMatrixRequestCreate
	if err = ctx.Bind(&req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData, err = h.ServicesApprovalMatrix.Create(ctx.Request().Context(), req)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Update is PUT /v1/approval-matrix/:id (answer has no data, as in
// site-dev-api).
func (h ApprovalMatrixHandler) Update(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var id uint64
	if id, err = strconv.ParseUint(ctx.GetUriParamString("id"), 10, 64); err != nil {
		err = familabs.ErrorInvalid("id")
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var req dto.ApprovalMatrixRequestUpdate
	if err = ctx.Bind(&req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	if err = h.ServicesApprovalMatrix.Update(ctx.Request().Context(), uint(id), req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}
