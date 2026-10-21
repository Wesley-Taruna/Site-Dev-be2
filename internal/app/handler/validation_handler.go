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

// ValidationHandler is /v1/validation: token check, and the approval
// e-mail link login.
type ValidationHandler struct {
	Option             global.HandlerOptions
	ServicesValidation service.IValidationService
}

// URLMapping implements router.RouteHandlers
func (h *ValidationHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesValidation = service.NewValidationService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *ValidationHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.POST("", h.Token, mw.Authorized())
	// no token here: the approval code is what is exchanged for one
	r.POST("/approval", h.Approval)
}

// Token is POST /v1/validation: is the token still good, and whose is it.
func (h ValidationHandler) Token(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesValidation.ValidateToken(ctx.Request().Context(), userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Approval is POST /v1/validation/approval {approval_code}.
func (h ValidationHandler) Approval(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.ApprovalCodeValidateRequest
	if err = ctx.Bind(&req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData, err = h.ServicesValidation.ValidateApprovalCode(ctx.Request().Context(), req.ApprovalCode)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}
