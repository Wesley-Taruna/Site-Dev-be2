package dto

// WilayahRequestGet is the query of GET /v1/wilayah/search: q (village
// name, at least 3 characters) or postal_code (exact).
type WilayahRequestGet struct {
	ListRequestGet
	PostalCode string
}

// WilayahSearchResponseGet is one village with district, regency and
// province (site-dev-api WilayahSearchResp).
type WilayahSearchResponseGet struct {
	VillageCode  string `json:"village_code"`
	VillageName  string `json:"village_name"`
	VillageType  string `json:"village_type"`
	PostalCode   string `json:"postal_code"`
	DistrictCode string `json:"district_code"`
	DistrictName string `json:"district_name"`
	RegencyCode  string `json:"regency_code"`
	RegencyName  string `json:"regency_name"`
	ProvinceCode string `json:"province_code"`
	ProvinceName string `json:"province_name"`
}

// ProvinceResponseGet is one province of the picker list.
type ProvinceResponseGet struct {
	ProvinceCode string `json:"province_code"`
	ProvinceName string `json:"province_name"`
}

// ProvinceTreeResponseGet is GET /v1/wilayah/provinces/:code: the province
// with every regency, district and village (site-dev-api ProvinceTreeResp).
type ProvinceTreeResponseGet struct {
	ProvinceCode string            `json:"province_code"`
	ProvinceName string            `json:"province_name"`
	Regencies    []RegencyTreeItem `json:"regencies"`
}

type RegencyTreeItem struct {
	RegencyCode string             `json:"regency_code"`
	RegencyName string             `json:"regency_name"`
	Districts   []DistrictTreeItem `json:"districts"`
}

type DistrictTreeItem struct {
	DistrictCode string        `json:"district_code"`
	DistrictName string        `json:"district_name"`
	Villages     []VillageItem `json:"villages"`
}

type VillageItem struct {
	VillageCode string `json:"village_code"`
	VillageName string `json:"village_name"`
	VillageType string `json:"village_type"`
	PostalCode  string `json:"postal_code"`
}
