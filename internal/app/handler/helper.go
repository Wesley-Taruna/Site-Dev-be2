package handler

import (
	"strings"

	appconstants "site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"

	"gitlab.com/fami-app-v2/familib/familabs"
	"gitlab.com/fami-app-v2/familib/familabs/constants"
)

// listRequest reads page, sort_by, sort_dir and q of a paged list.
func listRequest(ctx *familabs.Context) (req dto.ListRequestGet, page *familabs.Paginator, err error) {
	page, err = familabs.NewPaginator(ctx)
	if err != nil {
		return
	}

	req = dto.ListRequestGet{
		Page:    page.Page,
		SortBy:  ctx.GetParamString("sort_by"),
		SortDir: ctx.GetParamString("sort_dir"),
		Search:  ctx.GetParamString("q"),
	}
	return
}

// siteIDParam reads a site/task ID from the URL path: a slashed site ID of
// 27 Oct ("SD27/10/2026001") arrives as its twin "SD27-10-2026001" or
// "%2F"-encoded (Echo keeps path parameters escaped) and is turned back into
// the stored ID (constants.SiteIDFromPath). Current IDs ("SD28102026001"),
// old "ST…" IDs and task IDs pass unchanged.
func siteIDParam(ctx *familabs.Context, name string) string {
	return appconstants.SiteIDFromPath(ctx.GetUriParamString(name))
}

// siteIDQuery is siteIDParam for a query parameter (already decoded by Go;
// the twin is accepted too).
func siteIDQuery(ctx *familabs.Context, name string) string {
	v := ctx.QueryParam(name)
	if v == "" {
		return ""
	}
	return appconstants.SiteIDFromPath(v)
}

// userIDFromContext reads the user ID that middlewarex.Authorization (used
// by Middleware.Authorized) stores in the request context.
func userIDFromContext(ctx *familabs.Context) int64 {
	userID, _ := ctx.Request().Context().Value(constants.KeyUserID).(int64)
	return userID
}

// splitList reads "A,B" into [A B] (blanks dropped; "" = nil).
func splitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Page size of the November lists (Activity Log, portal): limit (or
// per_page), default 20, at most 100. The older site-dev lists keep their
// fixed 20 (dto.ListPageSize).
const (
	newListDefaultLimit = 20
	newListMaxLimit     = 100
)

// newListPage is the familabs paginator with the page size read from
// limit or per_page.
func newListPage(ctx *familabs.Context) (page *familabs.Paginator, err error) {
	if page, err = familabs.NewPaginator(ctx); err != nil {
		return
	}
	limit := ctx.GetParamInt("limit")
	if limit <= 0 {
		limit = page.PerPage
	}
	if limit <= 0 {
		limit = newListDefaultLimit
	}
	if limit > newListMaxLimit {
		limit = newListMaxLimit
	}
	page.PerPage, page.Limit = limit, limit
	page.Offset = (page.Page - 1) * limit
	page.Start, page.End = page.Offset, page.Offset+limit
	return
}
