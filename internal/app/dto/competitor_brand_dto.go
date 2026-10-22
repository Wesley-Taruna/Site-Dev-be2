package dto

// CompetitorBrandResponseGet is one row of GET /v1/competitor-brands
// (API-CHANGES §12; prototype src/lib/brands.ts). logo_url is LogoPath as
// stored (object-storage links wait for the storage plug, plan §7).
type CompetitorBrandResponseGet struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Patterns    []string `json:"patterns"`
	LogoURL     string   `json:"logo_url"`
	Color       string   `json:"color"`
	Type        string   `json:"type"`
	LegacyType  string   `json:"legacy_type"`
	Active      bool     `json:"active"`
	Placeholder bool     `json:"placeholder"`
	Sort        int      `json:"sort"`
}
