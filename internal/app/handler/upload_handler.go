package handler

import (
	"fmt"

	"site-dev-backend/global"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/middleware"
	"site-dev-backend/internal/app/service"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs"
	"gitlab.com/fami-app-v2/familib/familabs/log"
)

// UploadHandler is /v1/upload: site photos into file storage.
type UploadHandler struct {
	Option         global.HandlerOptions
	ServicesUpload service.IUploadService
}

// URLMapping implements router.RouteHandlers
func (h *UploadHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesUpload = service.NewUploadService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *UploadHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.POST("/add", h.Add, mw.Authorized())
	r.POST("/remove", h.Remove, mw.Authorized())
}

// Add is POST /v1/upload/add (multipart field "file").
func (h UploadHandler) Add(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	fh, formErr := ctx.FormFile("file")
	if formErr != nil {
		return ctx.Serve(familabs.ErrorInvalid("file"))
	}
	src, err := fh.Open()
	if err != nil {
		err = fmt.Errorf("open uploaded file: %w", err)
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}
	defer src.Close()

	ctx.ResponseData, err = h.ServicesUpload.Add(ctx.Request().Context(), fh.Filename, src)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Remove is POST /v1/upload/remove {"path": "<bucket>/<object...>"}.
func (h UploadHandler) Remove(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.UploadRemoveRequest
	if err = ctx.Bind(&req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	if err = h.ServicesUpload.Remove(ctx.Request().Context(), req.Path); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}
