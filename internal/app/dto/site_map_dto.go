package dto

// SiteMapResponseGet is one point of the map / one row of the "Site Active"
// table (site-dev-api SiteMapResp).
type SiteMapResponseGet struct {
	StoreCode      string  `json:"store_code"`
	StoreName      string  `json:"store_name"`
	StoreTypeName  string  `json:"store_type_name"`
	SiteType       string  `json:"site_type"`
	Longitude      float64 `json:"longitude"`
	Latitude       float64 `json:"latitude"`
	OpeningStoreDT *string `json:"opening_store_dt"`
	// TaskStatus is the stored task status of a "Site Requested" point
	// (November scope, API-CHANGES §4 / review #6); left out for stores.
	TaskStatus string `json:"task_status,omitempty"`
}

// SiteMapStoreTypeCount is the number of map points per store type
// (site-dev-api SiteMapStoreTypeCount).
type SiteMapStoreTypeCount struct {
	StoreTypeName string `json:"store_type_name"`
	Total         int    `json:"total"`
}

// SiteMapListResponseGet is GET /v1/site/maps (site-dev-api SiteMapListResp).
type SiteMapListResponseGet struct {
	Sites      []SiteMapResponseGet    `json:"sites"`
	StoreTypes []SiteMapStoreTypeCount `json:"store_types"`
}

// SiteRequestGet is the query of GET /v1/site.
type SiteRequestGet struct {
	ListRequestGet
	City   string
	Region string
}
