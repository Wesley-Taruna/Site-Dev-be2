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

// WilayahHandler is /v1/wilayah: the administrative regions (province >
// regency > district > village).
type WilayahHandler struct {
	Option          global.HandlerOptions
	ServicesWilayah service.IWilayahService
}

// URLMapping implements router.RouteHandlers
func (h *WilayahHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesWilayah = service.NewWilayahService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *WilayahHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.GET("/search", h.Search, mw.Authorized())
	r.GET("/provinces", h.Provinces, mw.Authorized())
	r.GET("/provinces/:code", h.ProvinceTree, mw.Authorized())
}

// Search is GET /v1/wilayah/search?q= (village name, 3+ characters) or
// ?postal_code= (exact). No q and no postal_code = empty list.
func (h WilayahHandler) Search(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var list dto.ListRequestGet
	var page *familabs.Paginator
	list, page, err = listRequest(ctx)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}
	req := dto.WilayahRequestGet{ListRequestGet: list, PostalCode: ctx.GetParamString("postal_code")}

	var rows []dto.WilayahSearchResponseGet
	var total int64
	switch {
	case req.PostalCode != "":
		rows, total, err = h.ServicesWilayah.SearchByPostalCode(ctx.Request().Context(), req.PostalCode, req.Page)
	case req.Search == "":
		// form just opened, nothing typed yet: empty list, not an error
		rows = []dto.WilayahSearchResponseGet{}
	case len(req.Search) < 3:
		err = familabs.ErrorInvalid("q (minimal 3 karakter)")
		return ctx.Serve(err)
	default:
		rows, total, err = h.ServicesWilayah.SearchByName(ctx.Request().Context(), req.Search, req.Page)
	}
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.DataList(rows, total, page)

	return ctx.Serve(err)
}

// Provinces is GET /v1/wilayah/provinces.
func (h WilayahHandler) Provinces(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesWilayah.GetProvinces(ctx.Request().Context())
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// ProvinceTree is GET /v1/wilayah/provinces/:code.
func (h WilayahHandler) ProvinceTree(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesWilayah.GetProvinceTree(ctx.Request().Context(), ctx.GetUriParamString("code"))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}
