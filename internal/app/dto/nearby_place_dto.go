package dto

import "time"

// Nearby places around a point (local §6): named competitor chains,
// offices, schools, campuses and apartments within a radius. Same JSON as
// site-dev-api local commit 2191e07 (and the prototype's AreaResp).

// NearbyPlaceItem is one named place.
type NearbyPlaceItem struct {
	// ID is stable per source: "g:<place_id>" or "o:<type>/<id>".
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	// Brand is set for competitors.
	Brand     string  `json:"brand,omitempty"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	DistanceM int     `json:"distance_m"`
	Source    string  `json:"source"`
	// PlaceID is the Google place id (for an exact Google Maps link).
	PlaceID string `json:"place_id,omitempty"`
	// Address (Wes, 29 Oct): short street address when the source has one
	// (Google shortFormattedAddress; OSM addr:* tags); left out otherwise.
	// Areas cached before 29 Oct have none until they expire.
	Address string `json:"address,omitempty"`
}

// NearbyCategoryInfo is the count and source of one category. Capped: the
// source cut the result, the real number may be higher ("≥ N"); the list
// is complete up to CompleteWithinM.
type NearbyCategoryInfo struct {
	Count           int    `json:"count"`
	Source          string `json:"source"`
	Capped          bool   `json:"capped"`
	CompleteWithinM *int   `json:"complete_within_m,omitempty"`
}

// NearestStoreResponse is the nearest active FamilyMart (AMFIBI_MASTER).
type NearestStoreResponse struct {
	Name      string `json:"name"`
	Code      string `json:"code"`
	DistanceM int    `json:"distance_m"`
	Within1Km int    `json:"within_1km"`
}

// NearbyAreaResponseGet is GET /v1/site/nearby-places.
type NearbyAreaResponseGet struct {
	Source            string                        `json:"source"`
	Lat               float64                       `json:"lat"`
	Lng               float64                       `json:"lng"`
	RadiusM           int                           `json:"radius_m"`
	FetchedAt         time.Time                     `json:"fetched_at"`
	Places            []NearbyPlaceItem             `json:"places"`
	Categories        map[string]NearbyCategoryInfo `json:"categories"`
	GoogleCalls       int                           `json:"google_calls"`
	Cached            bool                          `json:"cached"`
	FallbackReason    string                        `json:"fallback_reason,omitempty"`
	NearestFamilyMart *NearestStoreResponse         `json:"nearest_familymart"`
	// StoreSource is always "real" (MStore).
	StoreSource string `json:"store_source"`
	Attribution string `json:"attribution"`
}

// NearbyMeasured is the map data given to Gemini ("DATA TERUKUR") and
// returned with a fresh insight: full counts, every competitor, the nearest
// few of each other category.
type NearbyMeasured struct {
	Source            string                `json:"source"`
	RadiusM           int                   `json:"radius_m"`
	Counts            map[string]int        `json:"counts"`
	Capped            []string              `json:"capped,omitempty"`
	ExamplesPerGroup  int                   `json:"examples_per_group"`
	Items             []NearbyMeasuredItem  `json:"items"`
	NearestFamilyMart *NearestStoreResponse `json:"nearest_familymart"`
	StoreSource       string                `json:"store_source"`
	FetchedAt         time.Time             `json:"fetched_at"`
}

// NearbyMeasuredItem is one example place in NearbyMeasured.
type NearbyMeasuredItem struct {
	Name      string `json:"name"`
	Category  string `json:"category"`
	Brand     string `json:"brand,omitempty"`
	DistanceM int    `json:"distance_m"`
}

// NearbyPlacesRequestGet is the query of GET /v1/site/nearby-places.
type NearbyPlacesRequestGet struct {
	Lat     float64
	Lng     float64
	RadiusM int
	// Include: extra groups asked for (?include=other_store, Wes 29 Oct);
	// empty = the answer as before.
	Include []string
}
