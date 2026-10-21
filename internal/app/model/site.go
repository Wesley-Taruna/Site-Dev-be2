package model

import (
	"encoding/json"
	"time"
)

type Site struct {
	SiteID                 string     `gorm:"column:SiteID;primaryKey" json:"site_id"`
	SiteName1              string     `gorm:"column:SiteName1" json:"site_name_1"`
	SiteName2              string     `gorm:"column:SiteName2" json:"site_name_2"`
	SiteAddress1           string     `gorm:"column:SiteAddress1" json:"site_address_1"`
	SiteAddress2           string     `gorm:"column:SiteAddress2" json:"site_address_2"`
	SiteAddress3           string     `gorm:"column:SiteAddress3" json:"site_address_3"`
	SitePostalCode         string     `gorm:"column:SitePostalCode" json:"site_postal_code"`
	SiteDistrict           string     `gorm:"column:SiteDistrict" json:"site_district"`
	SiteCity               string     `gorm:"column:SiteCity" json:"site_city"`
	SiteRegion             string     `gorm:"column:SiteRegion" json:"site_region"`
	SiteLat                float64    `gorm:"column:SiteLat" json:"site_lat"`
	SiteLon                float64    `gorm:"column:SiteLon" json:"site_lon"`
	SiteTelepon            string     `gorm:"column:SiteTelepon" json:"site_telepon"`
	SiteTeleponExt         string     `gorm:"column:SiteTeleponExt" json:"site_telepon_ext"`
	SiteMobilePhone        string     `gorm:"column:SiteMobilePhone" json:"site_mobile_phone"`
	SitePic                string     `gorm:"column:SitePic" json:"site_pic"`
	SiteFax                string     `gorm:"column:SiteFax" json:"site_fax"`
	SitePlannedOpeningDate time.Time  `gorm:"column:SitePlannedOpeningDate" json:"site_planned_opening_date"`
	SiteSalesArea          float64    `gorm:"column:SiteSalesArea" json:"site_sales_area"`
	SiteSellingAlcohol     bool       `gorm:"column:SiteSellingAlcohol" json:"site_selling_alcohol"`
	SiteOpen24Hours        bool       `gorm:"column:SiteOpen24Hours" json:"site_open_24_hours"`
	CreatedBy              int64      `gorm:"column:CreatedBy" json:"created_by"`
	UpdatedBy              *int64     `gorm:"column:UpdatedBy" json:"updated_by"`
	CreatedAt              time.Time  `gorm:"column:CreatedAt" json:"created_at"`
	UpdatedAt              *time.Time `gorm:"column:UpdatedAt" json:"updated_at"`
}

func (Site) TableName() string {
	return "Sites"
}

func (m *Site) MarshalJSON() ([]byte, error) {
	type Alias Site

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}
