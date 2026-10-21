package handler

import (
	"mime"
	"net/http"
	"path/filepath"
	"strconv"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/middleware"
	"site-dev-backend/internal/app/service"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs"
	"gitlab.com/fami-app-v2/familib/familabs/log"
)

// AttachmentHandler is /v1/attachment: streams a site photo (the bucket is
// private, so the app fetches through here with its token).
type AttachmentHandler struct {
	Option         global.HandlerOptions
	ServicesUpload service.IUploadService
}

// URLMapping implements router.RouteHandlers
func (h *AttachmentHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesUpload = service.NewUploadService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *AttachmentHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.GET("/:id/file", h.File, mw.Authorized())
}

// File is GET /v1/attachment/:id/file: the raw file, content type from the
// file name; unknown attachment or missing file = 422 "not found" as in
// site-dev-api.
func (h AttachmentHandler) File(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	id, parseErr := strconv.ParseUint(ctx.GetUriParamString("id"), 10, 64)
	if parseErr != nil {
		return ctx.Serve(familabs.ErrorInvalid("id"))
	}

	attachment, reader, err := h.ServicesUpload.OpenAttachment(ctx.Request().Context(), uint(id))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		if attachment == nil {
			return ctx.Serve(familabs.ErrorNotFound("attachment"))
		}
		return ctx.Serve(familabs.ErrorNotFound("attachment file"))
	}
	defer reader.Close()

	contentType := mime.TypeByExtension(filepath.Ext(attachment.FileName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return ctx.Stream(http.StatusOK, contentType, reader)
}
