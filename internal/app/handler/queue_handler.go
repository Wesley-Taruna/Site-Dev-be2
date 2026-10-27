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

// QueueHandler is /v1/queue: callbacks of the background jobs (called by
// the job service, not by users), behind Basic auth.
type QueueHandler struct {
	Option            global.HandlerOptions
	ServicesQueue     service.IQueueService
	ServicesAIInsight service.IAIInsightService
}

// URLMapping implements router.RouteHandlers
func (h *QueueHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesQueue = service.NewQueueService()
	h.ServicesAIInsight = service.NewAIInsightService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *QueueHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.POST("/approval-email", h.ApprovalEmail, mw.BasicAuth())
	r.POST("/ai-insight", h.AIInsight, mw.BasicAuth())
	r.POST("/ai-insight-quota-flush", h.AIInsightQuotaFlush, mw.BasicAuth())
}

// ApprovalEmail is POST /v1/queue/approval-email: sends the queued
// approval e-mail.
func (h QueueHandler) ApprovalEmail(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.ApprovalEmailTaskPayload
	if err = ctx.Bind(&req); err != nil {
		return ctx.Serve(err)
	}

	if err = h.ServicesQueue.SendApprovalEmail(ctx.Request().Context(), req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// AIInsight is POST /v1/queue/ai-insight: generates the queued AI insight.
func (h QueueHandler) AIInsight(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.AIInsightTaskPayload
	if err = ctx.Bind(&req); err != nil {
		return ctx.Serve(err)
	}

	if _, err = h.ServicesAIInsight.Generate(ctx.Request().Context(), req.SiteID, req.ActorID); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		// nothing to do (empty survey, Wes 27 Oct): 200, so the queue does
		// not retry a job that can never succeed
		if service.AIInsightSkipped(err) {
			return ctx.Serve(nil)
		}
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// AIInsightQuotaFlush is POST /v1/queue/ai-insight-quota-flush (daily at
// 00:00 Asia/Jakarta): every user's AI insight counter is reset.
func (h QueueHandler) AIInsightQuotaFlush(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var deleted int
	if deleted, err = h.ServicesAIInsight.FlushQuota(ctx.Request().Context()); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData = map[string]int{"deleted": deleted}

	return ctx.Serve(err)
}
