package model

import (
	"encoding/json"
	"time"
)

// CompetitorBrand is one competitor chain (Indomaret, Alfamart, ...; the
// rest is "Lainnya"). November scope, API-CHANGES §12: new table
// CompetitorBrands, needs DB owner (plan §8). Patterns is a JSON array of
// case-insensitive regexes, checked in Sort order. SearchQuery (23 Oct) is
// the Google Text Search query of an ACTIVE brand ("" = no own search,
// found by the convenience_store Nearby Search, e.g. K3 Mart).
type CompetitorBrand struct {
	Slug          string     `gorm:"column:Slug;primaryKey" json:"slug"`
	Name          string     `gorm:"column:Name" json:"name"`
	Patterns      string     `gorm:"column:Patterns" json:"patterns"`
	SearchQuery   string     `gorm:"column:SearchQuery" json:"search_query"`
	LogoPath      string     `gorm:"column:LogoPath" json:"logo_path"`
	Color         string     `gorm:"column:Color" json:"color"`
	Type          string     `gorm:"column:Type" json:"type"`
	LegacyType    string     `gorm:"column:LegacyType" json:"legacy_type"`
	Active        bool       `gorm:"column:Active" json:"active"`
	IsPlaceholder bool       `gorm:"column:IsPlaceholder" json:"is_placeholder"`
	Sort          int        `gorm:"column:Sort" json:"sort"`
	UpdatedBy     *int64     `gorm:"column:UpdatedBy" json:"updated_by"`
	UpdatedAt     *time.Time `gorm:"column:UpdatedAt" json:"updated_at"`
}

func (CompetitorBrand) TableName() string {
	return "CompetitorBrands"
}

func (m *CompetitorBrand) MarshalJSON() ([]byte, error) {
	type Alias CompetitorBrand

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}
