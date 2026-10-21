package dto

// NearbyRequestResponseGet is one active request near a point: GET
// /v1/task/nearby, and the data of a 409 DUPLICATE_NEARBY on submit (local
// §8). Only these fields: no phone numbers, PIC or survey of other staff.
type NearbyRequestResponseGet struct {
	SiteID     string  `json:"site_id"`
	SiteName   string  `json:"site_name"`
	TaskStatus string  `json:"task_status"`
	OwnerID    int64   `json:"owner_id"`
	OwnerName  string  `json:"owner_name"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	DistanceM  int     `json:"distance_m"`
}

// NearbyRequestGet is the query of GET /v1/task/nearby.
type NearbyRequestGet struct {
	Lat           float64
	Lng           float64
	RadiusM       int
	ExcludeSiteID string
}
