package service

import (
	"fmt"
	"strings"

	dto "site-dev-backend/internal/app/dto"
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
	if page == 0 {
		page = 1
	}
	return (page - 1) * dto.ListPageSize
}
