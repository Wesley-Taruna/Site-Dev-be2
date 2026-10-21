package handler

import (
	"net/http"

	"fami-boilerplate-service/global"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs"
)

type HealthCheckHandler struct {
	Option global.HandlerOptions
}

// URLMapping implements router.RouteHandlers
func (h *HealthCheckHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	r.GET("", h.Index)
}

func (h HealthCheckHandler) Index(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)
	ctx.Message(http.StatusOK, "success", h.Option.Common.Config.App.Name)
	return
}
