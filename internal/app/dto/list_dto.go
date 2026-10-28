package dto

// ListRequestGet holds the common query of the paged lists: page (from the
// familabs paginator), sort_by, sort_dir and q. The page size is
// ListPageSize unless Limit is set.
type ListRequestGet struct {
	Page    int
	SortBy  string
	SortDir string
	Search  string
	// Limit: rows per page from limit / per_page (simple app, Wes 28 Oct;
	// pending-approvals and approval-history only). 0 = ListPageSize.
	Limit int
}
