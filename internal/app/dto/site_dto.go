package dto

import "time"

// SiteResponseGet is one site. JSON names match site-dev-api's SiteResp
// exactly (created_by_name/updated_by_name are omitted when empty, as there).
type SiteResponseGet struct {
	SiteID                 string     `json:"site_id"`
	SiteName1              string     `json:"site_name_1"`
	SiteName2              string     `json:"site_name_2"`
	SiteAddress1           string     `json:"site_address_1"`
	SiteAddress2           string     `json:"site_address_2"`
	SiteAddress3           string     `json:"site_address_3"`
	SitePostalCode         string     `json:"site_postal_code"`
	SiteDistrict           string     `json:"site_district"`
	SiteCity               string     `json:"site_city"`
	SiteRegion             string     `json:"site_region"`
	SiteLat                float64    `json:"site_lat"`
	SiteLon                float64    `json:"site_lon"`
	SiteTelepon            string     `json:"site_telepon"`
	SiteTeleponExt         string     `json:"site_telepon_ext"`
	SiteMobilePhone        string     `json:"site_mobile_phone"`
	SitePic                string     `json:"site_pic"`
	SiteFax                string     `json:"site_fax"`
	SitePlannedOpeningDate DateOnly   `json:"site_planned_opening_date"`
	SiteSalesArea          float64    `json:"site_sales_area"`
	SiteSellingAlcohol     bool       `json:"site_selling_alcohol"`
	SiteOpen24Hours        bool       `json:"site_open_24_hours"`
	CreatedBy              int64      `json:"created_by"`
	CreatedByName          string     `json:"created_by_name,omitempty"`
	UpdatedBy              *int64     `json:"updated_by"`
	UpdatedByName          string     `json:"updated_by_name,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              *time.Time `json:"updated_at"`
}
