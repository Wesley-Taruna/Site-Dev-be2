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

type SiteHandler struct {
	Option              global.HandlerOptions
	ServicesSite        service.ISiteService
	ServicesSiteContext service.ISiteContextService
	ServicesNearby      service.INearbyPlaceService
}

// URLMapping implements router.RouteHandlers
func (h *SiteHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesSite = service.NewSiteService()
	h.ServicesSiteContext = service.NewSiteContextService()
	h.ServicesNearby = service.NewNearbyPlaceService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *SiteHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.GET("", h.Index, mw.Authorized())
	r.GET("/maps", h.Maps, mw.Authorized())
	r.GET("/active-list", h.ActiveList, mw.Authorized())
	r.GET("/nearby-places", h.NearbyPlaces, mw.Authorized())
	r.GET("/:id", h.Detail, mw.Authorized())
	r.GET("/:id/context", h.Context, mw.Authorized())
	r.DELETE("/:id", h.DeleteDraft, mw.Authorized())
}

// DeleteDraft is DELETE /v1/site/:id (Wes, 26 Oct, "Hapus draft"): the
// creator removes a draft that was never sent; id = site_id or task_id.
func (h SiteHandler) DeleteDraft(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	if err = h.ServicesSite.DeleteDraft(ctx.Request().Context(), siteIDParam(ctx, "id"), userIDFromContext(ctx)); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Index is GET /v1/site: sites, filters city and region.
func (h SiteHandler) Index(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var list dto.ListRequestGet
	var page *familabs.Paginator
	list, page, err = listRequest(ctx)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}
	req := dto.SiteRequestGet{
		ListRequestGet: list,
		City:           ctx.GetParamString("city"),
		Region:         ctx.GetParamString("region"),
	}

	var sites []dto.SiteResponseGet
	var total int64
	sites, total, err = h.ServicesSite.Get(ctx.Request().Context(), req)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.DataList(sites, total, page)

	return ctx.Serve(err)
}

// Maps is GET /v1/site/maps: active stores plus requested sites.
func (h SiteHandler) Maps(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesSite.GetMaps(ctx.Request().Context())
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// ActiveList is GET /v1/site/active-list: the "Site Active" table.
func (h SiteHandler) ActiveList(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.ListRequestGet
	var page *familabs.Paginator
	req, page, err = listRequest(ctx)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var stores []dto.SiteMapResponseGet
	var total int64
	stores, total, err = h.ServicesSite.GetActiveStores(ctx.Request().Context(), req)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.DataList(stores, total, page)

	return ctx.Serve(err)
}

// Detail is GET /v1/site/:id.
func (h SiteHandler) Detail(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesSite.GetByID(ctx.Request().Context(), siteIDParam(ctx, "id"))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Context is GET /v1/site/:id/context: the on-site survey.
func (h SiteHandler) Context(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesSiteContext.GetBySiteID(ctx.Request().Context(), siteIDParam(ctx, "id"))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// NearbyPlaces is GET /v1/site/nearby-places?lat=&lng=&radius= (local §6):
// named competitors, offices, schools, campuses and apartments around the
// point, with source and radius. All roles.
func (h SiteHandler) NearbyPlaces(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.NearbyPlacesRequestGet
	var parseErr error
	if req.Lat, parseErr = strconv.ParseFloat(ctx.QueryParam("lat"), 64); parseErr != nil {
		return ctx.Serve(familabs.ErrorInvalid("lat"))
	}
	if req.Lng, parseErr = strconv.ParseFloat(ctx.QueryParam("lng"), 64); parseErr != nil {
		return ctx.Serve(familabs.ErrorInvalid("lng"))
	}
	if v := ctx.QueryParam("radius"); v != "" {
		if req.RadiusM, parseErr = strconv.Atoi(v); parseErr != nil || req.RadiusM < 0 {
			return ctx.Serve(familabs.ErrorInvalid("radius"))
		}
	}

	ctx.ResponseData, err = h.ServicesNearby.GetArea(ctx.Request().Context(), req)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}
