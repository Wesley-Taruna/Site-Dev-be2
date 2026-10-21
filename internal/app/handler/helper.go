package handler

import (
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

// userIDFromContext reads the user ID that middlewarex.Authorization (used
// by Middleware.Authorized) stores in the request context.
func userIDFromContext(ctx *familabs.Context) int64 {
	userID, _ := ctx.Request().Context().Value(constants.KeyUserID).(int64)
	return userID
}
