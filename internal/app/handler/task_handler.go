package handler

import (
	"errors"
	"net/http"
	"strconv"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/middleware"
	"site-dev-backend/internal/app/service"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs"
	"gitlab.com/fami-app-v2/familib/familabs/log"
)

type TaskHandler struct {
	Option              global.HandlerOptions
	ServicesTask        service.ITaskService
	ServicesSiteContext service.ISiteContextService
	ServicesAIInsight   service.IAIInsightService
	ServicesSite        service.ISiteService
	ServicesActivity    service.IActivityService
}

// URLMapping implements router.RouteHandlers
func (h *TaskHandler) URLMapping(r *echo.Group) {
	h.Option = global.Setup
	h.ServicesTask = service.NewTaskService()
	h.ServicesSiteContext = service.NewSiteContextService()
	h.ServicesAIInsight = service.NewAIInsightService()
	h.ServicesSite = service.NewSiteService()
	h.ServicesActivity = service.NewActivityService()

	h.routes(r, middleware.NewMiddleware())
}

// routes registers the endpoints (also used by the handler tests).
func (h *TaskHandler) routes(r *echo.Group, mw *middleware.Middleware) {
	r.GET("", h.Index, mw.Authorized())
	r.GET("/site-list", h.SiteList, mw.Authorized())
	r.GET("/pending-approvals", h.PendingApprovals, mw.Authorized())
	r.GET("/approval-history", h.ApprovalHistory, mw.Authorized())
	r.GET("/status/count", h.StatusCount, mw.Authorized())
	r.GET("/nearby", h.Nearby, mw.Authorized())
	r.GET("/approver-news", h.ApproverNews, mw.Authorized())
	r.POST("/approver-news/seen", h.ApproverNewsSeen, mw.Authorized())
	r.GET("/:id", h.Detail, mw.Authorized())
	r.GET("/:id/history", h.History, mw.Authorized())
	r.GET("/:id/attachment", h.Attachment, mw.Authorized())
	r.PUT("/context/:site_id", h.UpdateContext, mw.Authorized())
	r.POST("/ai-insight", h.GenerateAIInsight, mw.Authorized())
	r.POST("/ai-insight/preview", h.PreviewAIInsight, mw.Authorized())
	r.GET("/ai-insight/quota", h.AIInsightQuota, mw.Authorized())
	r.GET("/ai-insight/:site_id", h.AIInsight, mw.Authorized())
	r.POST("/:status", h.Save, mw.Authorized())
	r.POST("/:id/approve", h.Approve, mw.Authorized())
}

// Save is POST /v1/task/:status (DRAFT or SUBMITED) {site, context,
// attachment, notes, insight}: site.site_id empty = new site.
func (h TaskHandler) Save(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	status := ctx.GetUriParamString("status")
	if status != constants.TaskStatusDraft && status != constants.TaskStatusSubmited {
		return ctx.Serve(familabs.ErrorInvalid("status (must be DRAFT or SUBMITED)"))
	}

	var req dto.SiteRequestUpsert
	if err = ctx.Bind(&req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData, err = h.ServicesSite.UpsertWithStatus(ctx.Request().Context(), req, status, userIDFromContext(ctx))
	var dupErr *service.DuplicateNearbyError
	if errors.As(err, &dupErr) {
		// 409 with the nearby requests as data; the app shows them and may
		// send again with "confirm_duplicate": true (local §8)
		ctx.Data(dupErr.Items)
		return ctx.Serve(echo.NewHTTPError(http.StatusConflict, dupErr.Error()))
	}
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Nearby is GET /v1/task/nearby?lat=&lng=&radius=&exclude_site_id= (local
// §8): active requests of all staff near a point, nearest first.
func (h TaskHandler) Nearby(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	req := dto.NearbyRequestGet{ExcludeSiteID: ctx.QueryParam("exclude_site_id")}
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

	ctx.ResponseData, err = h.ServicesTask.GetNearbyRequests(ctx.Request().Context(), req)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Approve is POST /v1/task/:id/approve {decision, notes} (id = task_id).
// The answer has no data, as in site-dev-api.
func (h TaskHandler) Approve(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.TaskActionRequest
	if err = ctx.Bind(&req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	if err = h.ServicesTask.Approve(ctx.Request().Context(), ctx.GetUriParamString("id"), userIDFromContext(ctx), req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// GenerateAIInsight is POST /v1/task/ai-insight {site_id}: a new Gemini
// insight for a saved site (replaces the old one).
func (h TaskHandler) GenerateAIInsight(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.AIInsightRequestGenerate
	if err = ctx.Bind(&req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData, err = h.ServicesAIInsight.Generate(ctx.Request().Context(), req.SiteID, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// PreviewAIInsight is POST /v1/task/ai-insight/preview: insight for a site
// that is not saved yet; nothing is stored.
func (h TaskHandler) PreviewAIInsight(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.AIInsightRequestPreview
	if err = ctx.Bind(&req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData, err = h.ServicesAIInsight.Preview(ctx.Request().Context(), req, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// AIInsightQuota is GET /v1/task/ai-insight/quota: the caller's Gemini
// calls left today.
func (h TaskHandler) AIInsightQuota(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesAIInsight.GetQuota(ctx.Request().Context(), userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// AIInsight is GET /v1/task/ai-insight/:site_id: the stored insight.
func (h TaskHandler) AIInsight(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesAIInsight.Get(ctx.Request().Context(), ctx.GetUriParamString("site_id"))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// UpdateContext is PUT /v1/task/context/:site_id: replaces the on-site
// survey (creator only, DRAFT/REVISION only).
func (h TaskHandler) UpdateContext(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.SiteContextRequestUpsert
	if err = ctx.Bind(&req); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData, err = h.ServicesSiteContext.Update(ctx.Request().Context(), ctx.GetUriParamString("site_id"), req, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Index is GET /v1/task: the caller's task list (site staff see only their
// own tasks), 20 rows per page.
func (h TaskHandler) Index(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	// get params pagination
	var page *familabs.Paginator
	page, err = familabs.NewPaginator(ctx)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	req := dto.TaskRequestGet{
		Page:   page.Page,
		SiteID: ctx.GetParamString("site_id"),
		Status: ctx.GetParamString("status"),
		// November scope (open, review §5 #7): status_not=DRAFT
		StatusNot: splitList(ctx.GetParamString("status_not")),
		Search:    ctx.GetParamString("q"),
		SortBy:    ctx.GetParamString("sort_by"),
		SortDir:   ctx.GetParamString("sort_dir"),
	}

	var tasks []dto.TaskResponseGet
	var total int64
	tasks, total, err = h.ServicesTask.Get(ctx.Request().Context(), req, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.DataList(tasks, total, page)

	return ctx.Serve(err)
}

// History is GET /v1/task/:id/history (id = site_id): the task's history
// rows, oldest first.
func (h TaskHandler) History(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	siteID := ctx.GetUriParamString("id")

	var histories []dto.TaskHistoryResponseGet
	histories, err = h.ServicesTask.GetHistory(ctx.Request().Context(), siteID)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.ResponseData = histories

	return ctx.Serve(err)
}

// Detail is GET /v1/task/:id (id = task_id or site_id): site, on-site
// context, approval chain, history and attachments in one answer.
func (h TaskHandler) Detail(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesTask.GetDetail(ctx.Request().Context(), ctx.GetUriParamString("id"), userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// Attachment is GET /v1/task/:id/attachment (id = site_id): the site's
// attachments only.
func (h TaskHandler) Attachment(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesTask.GetAttachments(ctx.Request().Context(), ctx.GetUriParamString("id"))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// StatusCount is GET /v1/task/status/count: tasks per status (site staff:
// only their own).
func (h TaskHandler) StatusCount(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesTask.CountByStatus(ctx.Request().Context(), userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// SiteList is GET /v1/task/site-list: the "Site Request" table, every
// role sees every site.
func (h TaskHandler) SiteList(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.ListRequestGet
	var page *familabs.Paginator
	req, page, err = listRequest(ctx)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var tasks []dto.TaskResponseGet
	var total int64
	tasks, total, err = h.ServicesTask.GetSiteList(ctx.Request().Context(), req)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.DataList(tasks, total, page)

	return ctx.Serve(err)
}

// PendingApprovals is GET /v1/task/pending-approvals: what waits for the
// caller's decision.
func (h TaskHandler) PendingApprovals(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.ListRequestGet
	var page *familabs.Paginator
	req, page, err = listRequest(ctx)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var rows []dto.PendingApprovalResponseGet
	var total int64
	rows, total, err = h.ServicesTask.GetPendingApprovals(ctx.Request().Context(), req, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.DataList(rows, total, page)

	return ctx.Serve(err)
}

// ApprovalHistory is GET /v1/task/approval-history: what the caller
// already decided.
func (h TaskHandler) ApprovalHistory(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	var req dto.ListRequestGet
	var page *familabs.Paginator
	req, page, err = listRequest(ctx)
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	var rows []dto.ApprovalHistoryResponseGet
	var total int64
	rows, total, err = h.ServicesTask.GetApprovalHistory(ctx.Request().Context(), req, userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	ctx.DataList(rows, total, page)

	return ctx.Serve(err)
}

// ApproverNews is GET /v1/task/approver-news: what is new for an approver
// since the Persetujuan tab was last opened (November scope, prototype 21
// Oct).
func (h TaskHandler) ApproverNews(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	ctx.ResponseData, err = h.ServicesActivity.GetApproverNews(ctx.Request().Context(), userIDFromContext(ctx))
	if err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}

// ApproverNewsSeen is POST /v1/task/approver-news/seen.
func (h TaskHandler) ApproverNewsSeen(c echo.Context) (err error) {
	ctx := c.(*familabs.Context)

	if err = h.ServicesActivity.MarkApproverNewsSeen(ctx.Request().Context(), userIDFromContext(ctx)); err != nil {
		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
		return ctx.Serve(err)
	}

	return ctx.Serve(err)
}
