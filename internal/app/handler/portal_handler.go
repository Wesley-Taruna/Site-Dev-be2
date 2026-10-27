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

// PortalHandler is /v1/portal/site: the "Site Dev" menu of FamilyMart's web
// portal (November scope, API-CHANGES §11). How the portal logs in is open
// (review §5 #12): for now the same token check as every other route.
type PortalHandler struct {
	Option         global.HandlerOptions
	ServicesAccess service.IAccessService
	ServicesPortal service.IPortalService
}

// URLMapping implements router.RouteHandlers
func (h *PortalHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesAccess = service.NewAccessService()
	h.ServicesPortal = service.NewPortalService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *PortalHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.GET("", h.Index, mw.Authorized())
	r.GET("/status/count", h.StatusCount, mw.Authorized())
	r.GET("/:id/report", h.Report, mw.Authorized())
	r.PUT("/:id/ads", h.UpdateAds, mw.Authorized())
}

// Index is GET /v1/portal/site?status=<group>&q=&page=&limit=.
func (h PortalHandler) Index(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var page *familabs.Paginator
	if page, err = newListPage(ctx); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	req := dto.PortalSiteRequestGet{Status: ctx.GetParamString("status"), Search: ctx.GetParamString("q"), Offset: page.Offset, Limit: page.Limit}

	var rows []dto.PortalSiteResponseGet
	var total int64
	rows, total, err = h.ServicesPortal.Get(ctx.Request().Context(), req, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.DataList(rows, total, page)

	return ctx.Serve(err)
}

// StatusCount is GET /v1/portal/site/status/count: the 5 groups.
func (h PortalHandler) StatusCount(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesPortal.CountByGroup(ctx.Request().Context(), userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Report is GET /v1/portal/site/:id/report (id = site_id).
func (h PortalHandler) Report(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesPortal.GetReport(ctx.Request().Context(), siteIDParam(ctx, "id"), userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// UpdateAds is PUT /v1/portal/site/:id/ads {ads_value, ads_note}.
// DEFAULT editor: ADMIN (review §5 #11 open).
func (h PortalHandler) UpdateAds(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	if err = h.ServicesAccess.RequireAdmin(ctx.Request().Context(), userIDFromContext(ctx)); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var req dto.SiteAdsRequestUpdate
	if err = ctx.Bind(&req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData, err = h.ServicesPortal.UpdateAds(ctx.Request().Context(), siteIDParam(ctx, "id"), req, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}
