package model

// StoreMap is one row of the MStore/MStoreType query (AMFIBI_MASTER, another
// database on the same server). Not a table. OpeningStoreDT is a string on
// that side, so it is scanned as text.
type StoreMap struct {
	StoreCode      string  `gorm:"column:StoreCode"`
	StoreName      string  `gorm:"column:StoreName"`
	StoreTypeName  string  `gorm:"column:StoreTypeName"`
	SiteType       string  `gorm:"column:SiteType"`
	Longitude      float64 `gorm:"column:Longitude"`
	Latitude       float64 `gorm:"column:Latitude"`
	OpeningStoreDT *string `gorm:"column:OpeningStoreDT"`
}
