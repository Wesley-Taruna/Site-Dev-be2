package dto

// ReverseGeocodeResponse is GET /v1/geocoding/reverse (site-dev-api
// ReverseGeocodeResp): address lines and postal code of a point; district,
// city and region come from the wilayah lookup by postal code instead.
type ReverseGeocodeResponse struct {
	Address1   string `json:"address1"`
	Address2   string `json:"address2"`
	Address3   string `json:"address3"`
	PostalCode string `json:"postal_code"`
}

// AddressSearchResult is one hit of GET /v1/geocoding/search?q=
// (site-dev-api AddressSearchResult).
type AddressSearchResult struct {
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	DisplayName string  `json:"display_name"`
	Address1    string  `json:"address1"`
	Address2    string  `json:"address2"`
	Address3    string  `json:"address3"`
	PostalCode  string  `json:"postal_code"`
}
