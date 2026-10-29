package model

// FamilyMartStore is one active FamilyMart store with coordinates, read
// for GET /v1/site/familymart-stores (Wes, 29 Oct). Not a table: the
// columns come from [store_database] (default AMFIBI_MASTER.dbo.MStore,
// the same table /v1/site/maps and nearest_familymart read). Address is
// NULL when no address column is set or the store has none.
type FamilyMartStore struct {
	StoreCode string  `gorm:"column:StoreCode" json:"c"`
	StoreName string  `gorm:"column:StoreName" json:"n"`
	Address   *string `gorm:"column:Address" json:"a,omitempty"`
	Latitude  float64 `gorm:"column:Latitude" json:"la"`
	Longitude float64 `gorm:"column:Longitude" json:"lo"`
}
