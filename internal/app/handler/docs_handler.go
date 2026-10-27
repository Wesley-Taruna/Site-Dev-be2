package handler

import (
	"net/http"

	"site-dev-backend/docs"
	"site-dev-backend/global"

	"github.com/labstack/echo/v4"
)

// DocsHandler is /v1/docs: the interactive API page and docs/openapi.yaml.
// Registered only when config [docs] enabled = true (default false, so a
// production config without the key keeps them closed: 404). No login on
// these two routes; the page sends the Bearer token the reader types in
// with every "Try it out" call, and those calls go through the normal
// Authorized() check.
type DocsHandler struct {
	Option global.HandlerOptions
}

// URLMapping implements router.RouteHandlers
func (h *DocsHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	if !docsEnabled(h.Option) {
		return
	}

	h.routes(r)
}

// docsEnabled reads [docs] enabled (missing = false).
func docsEnabled(o global.HandlerOptions) bool {
	return o.Common.Env != nil && o.Common.Env.GetBool("docs.enabled")
}

// routes registers the endpoints (also used by the handler tests).
func (h *DocsHandler) routes(r *echo.Group) {
	r.GET("", h.Page)
	r.GET("/openapi.yaml", h.Spec)
}

// Page is GET /v1/docs: the Swagger UI page.
func (h DocsHandler) Page(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
	return c.HTMLBlob(http.StatusOK, docs.SwaggerUI)
}

// Spec is GET /v1/docs/openapi.yaml.
func (h DocsHandler) Spec(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
	return c.Blob(http.StatusOK, "application/yaml; charset=utf-8", docs.OpenAPI)
}
