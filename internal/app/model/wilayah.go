package model

import "time"

// MProvince, MRegency, MDistrict and MVillage are the administrative regions
// (KEPMENDAGRI 2025): province > regency (kota/kabupaten) > district
// (kecamatan) > village (kelurahan/desa).
type MProvince struct {
	ProvinceCode string     `gorm:"column:ProvinceCode;primaryKey" json:"province_code"`
	ProvinceName string     `gorm:"column:ProvinceName" json:"province_name"`
	CreatedAt    time.Time  `gorm:"column:CreatedAt" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"column:UpdatedAt" json:"updated_at"`
}

func (MProvince) TableName() string {
	return "MProvinces"
}

type MRegency struct {
	RegencyCode  string     `gorm:"column:RegencyCode;primaryKey" json:"regency_code"`
	RegencyName  string     `gorm:"column:RegencyName" json:"regency_name"`
	ProvinceCode string     `gorm:"column:ProvinceCode" json:"province_code"`
	CreatedAt    time.Time  `gorm:"column:CreatedAt" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"column:UpdatedAt" json:"updated_at"`
}

func (MRegency) TableName() string {
	return "MRegencies"
}

type MDistrict struct {
	DistrictCode string     `gorm:"column:DistrictCode;primaryKey" json:"district_code"`
	DistrictName string     `gorm:"column:DistrictName" json:"district_name"`
	RegencyCode  string     `gorm:"column:RegencyCode" json:"regency_code"`
	CreatedAt    time.Time  `gorm:"column:CreatedAt" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"column:UpdatedAt" json:"updated_at"`
}

func (MDistrict) TableName() string {
	return "MDistricts"
}

type MVillage struct {
	VillageCode  string     `gorm:"column:VillageCode;primaryKey" json:"village_code"`
	VillageName  string     `gorm:"column:VillageName" json:"village_name"`
	VillageType  string     `gorm:"column:VillageType" json:"village_type"`
	PostalCode   string     `gorm:"column:PostalCode" json:"postal_code"`
	DistrictCode string     `gorm:"column:DistrictCode" json:"district_code"`
	CreatedAt    time.Time  `gorm:"column:CreatedAt" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"column:UpdatedAt" json:"updated_at"`
}

func (MVillage) TableName() string {
	return "MVillages"
}

// WilayahSearchRow is one village with its district, regency and province
// (query result, not a table).
type WilayahSearchRow struct {
	VillageCode  string `gorm:"column:VillageCode"`
	VillageName  string `gorm:"column:VillageName"`
	VillageType  string `gorm:"column:VillageType"`
	PostalCode   string `gorm:"column:PostalCode"`
	DistrictCode string `gorm:"column:DistrictCode"`
	DistrictName string `gorm:"column:DistrictName"`
	RegencyCode  string `gorm:"column:RegencyCode"`
	RegencyName  string `gorm:"column:RegencyName"`
	ProvinceCode string `gorm:"column:ProvinceCode"`
	ProvinceName string `gorm:"column:ProvinceName"`
}
