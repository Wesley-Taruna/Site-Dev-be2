package dto

// FamilyMart stores near a point (Wes, 29 Oct): every active FamilyMart
// within 1 km of the user, to draw on the map (Pilih titik, detail map).

// FamilyMartStoresRequestGet is the query of GET /v1/site/familymart-stores.
type FamilyMartStoresRequestGet struct {
	Lat float64
	Lng float64
	// RadiusM 0 = [familymart_stores] radius_m (1000).
	RadiusM int
}

// FamilyMartStoreItem is one store, nearest first.
type FamilyMartStoreItem struct {
	StoreCode string `json:"store_code"`
	StoreName string `json:"store_name"`
	// Address is "" when the store database has no address for it (or no
	// address column is set in [store_database]).
	Address   string  `json:"address"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	DistanceM int     `json:"distance_m"`
}

// FamilyMartStoresResponseGet is GET /v1/site/familymart-stores.
type FamilyMartStoresResponseGet struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	RadiusM int     `json:"radius_m"`
	Count   int     `json:"count"`
	// Capped: more stores are inside the radius than [familymart_stores]
	// max_results; the farthest are left out.
	Capped bool `json:"capped"`
	// Cached: the store rows came from the server cache (Redis).
	Cached bool                  `json:"cached"`
	Stores []FamilyMartStoreItem `json:"stores"`
}
