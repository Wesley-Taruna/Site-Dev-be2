package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// orderClause builds the ORDER BY of a list the same way as site-dev-api's
// Pagination.SetOrderClause: defaultColumn unless sortBy is allowed;
// sort_dir desc/-1 = DESC, asc/1 or anything else = ASC.
func orderClause(sortBy string, sortDir string, defaultColumn string, allowed map[string]bool) string {
	column := defaultColumn
	if sortBy != "" && allowed[sortBy] {
		column = sortBy
	}

	direction := "ASC"
	switch strings.ToLower(sortDir) {
	case "desc", "-1":
		direction = "DESC"
	case "asc", "1":
		direction = "ASC"
	}

	return fmt.Sprintf("%s %s", column, direction)
}

// pageOffset is the row offset of a page of dto.ListPageSize rows (page 0
// counts as page 1), like site-dev-api's Pagination.GetOffset.
func pageOffset(page int) int {
	return pageOffsetOf(page, dto.ListPageSize)
}

// pageOffsetOf is pageOffset for a page of limit rows.
func pageOffsetOf(page int, limit int) int {
	if page == 0 {
		page = 1
	}
	return (page - 1) * limit
}

// pageLimit is the rows per page asked for (limit / per_page), or
// dto.ListPageSize when none was asked (the handler already caps it at 100).
func pageLimit(limit int) int {
	if limit <= 0 {
		return dto.ListPageSize
	}
	return limit
}

// getUser loads the caller from Users. The role always comes from there, not
// from the token (same as site-dev-api's Authenticated middleware): unknown
// user = 401 "user not found".
func getUser(ctx context.Context, repo repository.IUserRepository, userID int64) (user *model.User, err error) {
	user, err = repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = echo.NewHTTPError(http.StatusUnauthorized, "user not found")
		} else {
			err = echo.NewHTTPError(http.StatusInternalServerError, "failed to verify user")
		}
		return
	}
	return
}

// usernameOf returns a user's name, or "" when the user is unknown (the app
// then shows "User #<id>"), like site-dev-api's TaskService.usernameOf.
func usernameOf(ctx context.Context, repo repository.IUserRepository, userID int64) string {
	if userID == 0 {
		return ""
	}
	user, err := repo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return ""
	}
	return user.Username
}

// ignoreNotFound turns gorm.ErrRecordNotFound into nil (site-dev-api
// global.HandleGORMErr).
func ignoreNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return err
}
