package handler

import (
	"strconv"
	"strings"

	"site-dev-backend/global"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/middleware"
	"site-dev-backend/internal/app/service"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs"
	"gitlab.com/fami-app-v2/familib/familabs/log"
)

// GeocodingHandler is /v1/geocoding: Google Geocoding for the address
// fields (daily quota per user).
type GeocodingHandler struct {
	Option            global.HandlerOptions
	ServicesGeocoding service.IGeocodingService
}

// URLMapping implements router.RouteHandlers
func (h *GeocodingHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesGeocoding = service.NewGeocodingService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *GeocodingHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.GET("/reverse", h.Reverse, mw.Authorized())
	r.GET("/search", h.Search, mw.Authorized())
}

// Reverse is GET /v1/geocoding/reverse?lat=&lng=.
func (h GeocodingHandler) Reverse(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	lat, parseErr := strconv.ParseFloat(ctx.QueryParam("lat"), 64)
	if parseErr != nil {
		return ctx.Serve(familabs.ErrorInvalid("lat"))
	}
	lng, parseErr := strconv.ParseFloat(ctx.QueryParam("lng"), 64)
	if parseErr != nil {
		return ctx.Serve(familabs.ErrorInvalid("lng"))
	}

	ctx.ResponseData, err = h.ServicesGeocoding.Reverse(ctx.Request().Context(), lat, lng, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Search is GET /v1/geocoding/search?q= (3+ characters; empty = []).
func (h GeocodingHandler) Search(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	q := strings.TrimSpace(ctx.QueryParam("q"))
	if q == "" {
		ctx.ResponseData = []dto.AddressSearchResult{}
		return ctx.Serve(err)
	}
	if len(q) < 3 {
		return ctx.Serve(familabs.ErrorInvalid("q (minimal 3 karakter)"))
	}

	ctx.ResponseData, err = h.ServicesGeocoding.Search(ctx.Request().Context(), q, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}
