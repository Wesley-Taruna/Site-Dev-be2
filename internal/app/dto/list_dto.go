package dto

// ListRequestGet holds the common query of the paged lists: page (from the
// familabs paginator), sort_by, sort_dir and q. The page size is always
// ListPageSize.
type ListRequestGet struct {
	Page    int
	SortBy  string
	SortDir string
	Search  string
}
