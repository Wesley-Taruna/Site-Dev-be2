package handler

import (
	"site-dev-backend/global"
	"site-dev-backend/internal/app/middleware"
	"site-dev-backend/internal/app/service"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs"
	"gitlab.com/fami-app-v2/familib/familabs/log"
)

// CompetitorBrandHandler is /v1/competitor-brands (November scope,
// API-CHANGES §12). Brand admin (add/edit/logo) comes later, once the
// role is decided (review §5 #10).
type CompetitorBrandHandler struct {
	Option                  global.HandlerOptions
	ServicesCompetitorBrand service.ICompetitorBrandService
}

// URLMapping implements router.RouteHandlers
func (h *CompetitorBrandHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesCompetitorBrand = service.NewCompetitorBrandService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *CompetitorBrandHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.GET("", h.Index, mw.Authorized())
}

// Index is GET /v1/competitor-brands (every logged-in user).
func (h CompetitorBrandHandler) Index(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesCompetitorBrand.Get(ctx.Request().Context())
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}
