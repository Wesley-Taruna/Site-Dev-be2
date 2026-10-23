package client

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dagoLat, dagoLng = -6.8860, 107.6130

func TestBrandOf(t *testing.T) {
	for in, want := range map[string]string{
		"Indomaret Point": "Indomaret", "Alfamidi Super": "Alfamidi", "Alfa Express": "Alfamart", "Alfamart Sudirman": "Alfamart",
		"Circle-K": "Circle K", "K3Mart": "K3 Mart",
		// 23 Oct (Wes): 7-Eleven, Lulu Mart out, Yomart off; coffee chains in
		"7-Eleven": "", "Lulu Mart Tebet": "", "Yomart Kuningan": "",
		"Kopi Kenangan - Menara Rajawali": "Kopi Kenangan", "KopiKenangan": "Kopi Kenangan", "TOMORO COFFEE Kuningan": "Tomoro Coffee", "Tomoro": "Tomoro Coffee", "Kenangan Coffee": "", "Warung Madura": "", "Circlekebab Corner": "",
		// API Lab 22 Oct spelling variants
		"Indomart Simpang": "Indomaret", "Indo Maret Dago": "Indomaret", "Alfa Expres Dago": "Alfamart",
	} {
		assert.Equal(t, want, BrandOf(in), in)
	}
	assert.True(t, IsNotAShop("BII ATM Indomaret Karbela"))
	assert.True(t, IsNotAShop("Gudang Alfamart"))
	assert.False(t, IsNotAShop("Indomaret Dago Atas"))
	assert.True(t, IsNotAShopType("atm"))
	assert.False(t, IsNotAShopType("convenience_store"))
}

func countBy(places []dto.NearbyPlaceItem) map[string]int {
	out := map[string]int{}
	for _, p := range places {
		out[p.Category]++
	}
	return out
}

// Same expectations as site-dev-api 2191e07 TestParseOverpassArea (same
// fixture, copied unchanged).
func TestParseOverpassArea(t *testing.T) {
	raw, err := os.ReadFile("testdata/overpass_dago.json")
	require.NoError(t, err)

	places, err := ParseOverpassArea(raw, dagoLat, dagoLng, 620, DefaultBrandSet())

	require.NoError(t, err)
	assert.Equal(t, map[string]int{"competitor": 4, "office": 2, "school": 3, "university": 1, "apartment": 1}, countBy(places))
	for _, p := range places {
		assert.NotContains(t, p.Name, "FamilyMart", "own stores are never competitors")
		for _, bad := range []string{"Vaporline", "Superindo", "Pos Satpam", "Labtek IV", "Indomaret Jauh"} {
			assert.False(t, strings.Contains(p.Name, bad), "%q must be excluded", p.Name)
		}
		assert.Equal(t, constants.NearbySourceOSM, p.Source)
		assert.True(t, strings.HasPrefix(p.ID, "o:"))
	}
	for i := 1; i < len(places); i++ {
		assert.LessOrEqual(t, places[i-1].DistanceM, places[i].DistanceM)
	}
	assert.Equal(t, "Indomaret Fresh Dago", places[0].Name)
	assert.Equal(t, "Indomaret", places[0].Brand)
	assert.Equal(t, 101, places[0].DistanceM)

	_, err = ParseOverpassArea([]byte("<html>busy</html>"), dagoLat, dagoLng, 500, DefaultBrandSet())
	assert.Error(t, err)
	_, err = ParseOverpassArea([]byte(`{"remark":"runtime error"}`), dagoLat, dagoLng, 500, DefaultBrandSet())
	assert.Error(t, err)
}

func TestOverpassAreaQuery(t *testing.T) {
	q := OverpassAreaQuery(dagoLat, dagoLng, 500, DefaultBrandSet())
	assert.True(t, strings.HasPrefix(q, "[out:json][timeout:60][bbox:-6.89049,107.60848,-6.88151,107.61752];"), q[:80])
	assert.True(t, strings.HasSuffix(q, "out center tags 5000;"))
	names := `indo *-? *mare?t|alfa *mart|alfa *expres|alfa *midi|lawson|circle *-? *k|k3 *mart|kopi *kenangan|tomoro`
	assert.Contains(t, q, `nwr[shop][name~"`+names+`",i];`)
	assert.Contains(t, q, `nwr[amenity~"^(cafe|restaurant|fast_food)$"][name~"`+names+`",i];`)
	for _, gone := range []string{"eleven", "yo *mart", "lulu"} {
		assert.NotContains(t, q, gone)
	}
	assert.NotContains(t, OverpassAreaQuery(dagoLat, dagoLng, 500, BrandSet{}), "nwr[shop][name~", "no brand: no brand name lines")
}

func TestBrandSet(t *testing.T) {
	set, errs := NewBrandSet([]Brand{
		{Name: "Kopi Kenangan", Query: "Kopi Kenangan"},            // no pattern: the name
		{Name: "Broken", Query: "Broken", Patterns: []string{`(`}}, // skipped
		{Name: " ", Query: "x"},                                    // skipped
		{Name: "Tomoro Coffee", Query: " Tomoro Coffee ", Patterns: []string{`tomoro`, `(`}},
	})
	assert.Len(t, errs, 4)
	assert.Equal(t, []string{"Kopi Kenangan", "Tomoro Coffee"}, set.Names())
	assert.Equal(t, "Kopi Kenangan", set.Of("KOPI KENANGAN Dago"))
	assert.Equal(t, "Tomoro Coffee", set.Searched()[1].Name)
	assert.Equal(t, "Tomoro Coffee", set.Searched()[1].Query, "trimmed")
	assert.Len(t, set.Key(), 8)
	assert.NotEqual(t, DefaultBrandSet().Key(), set.Key(), "another list = another cache key")
	assert.Equal(t, DefaultBrandSet().Key(), DefaultBrandSet().Key())
	assert.Equal(t, "", BrandSet{}.Of("Indomaret"))
	assert.Equal(t, "", BrandSet{}.Key())
}

// OSM: branded cafés count (Kopi Kenangan as amenity=cafe, Tomoro as
// shop=coffee); an unbranded café, a 7-Eleven / Yomart shop = not a brand.
func TestOSMCategoryCoffeeAndInactive(t *testing.T) {
	b := DefaultBrandSet()
	for _, tc := range []struct {
		tags      map[string]string
		cat, want string
		ok        bool
	}{
		{map[string]string{"amenity": "cafe", "name": "Kopi Kenangan Menara"}, "competitor", "Kopi Kenangan", true},
		{map[string]string{"shop": "coffee", "name": "TOMORO COFFEE"}, "competitor", "Tomoro Coffee", true},
		{map[string]string{"amenity": "cafe", "name": "Kedai Kopi Bu Ani"}, "", "", false},
		{map[string]string{"shop": "convenience", "name": "7-Eleven Kuningan"}, "competitor", BrandOther, true},
		{map[string]string{"shop": "convenience", "name": "Yomart Tebet"}, "competitor", BrandOther, true},
		{map[string]string{"amenity": "atm", "name": "ATM Indomaret"}, "", "", false},
	} {
		cat, brand, ok := osmCategoryOf(tc.tags, b)
		assert.Equal(t, tc.ok, ok, tc.tags["name"])
		assert.Equal(t, tc.cat, cat, tc.tags["name"])
		assert.Equal(t, tc.want, brand, tc.tags["name"])
	}
}

// The local seed's ACTIVE CompetitorBrands rows = DefaultBrands (same list,
// order, queries and patterns), so the code fallback and the data agree.
func TestDefaultBrandsMatchSeed(t *testing.T) {
	raw, err := os.ReadFile("../../../scripts/localdev/sql/02_seed_master.sql")
	if os.IsNotExist(err) {
		t.Skip("local seed lives on the local-run branch")
	}
	require.NoError(t, err)
	row := regexp.MustCompile(`\('([a-z0-9-]+)',\s*N'([^']*)',\s*N'(\[[^']*\])',\s*N'([^']*)',.*?,\s*([01]),\s*[01],\s*(\d+)\)`)
	var active []Brand
	for _, m := range row.FindAllStringSubmatch(string(raw), -1) {
		if m[5] != "1" {
			continue
		}
		var patterns []string
		// SQL text holds JSON with \\ escapes
		require.NoError(t, json.Unmarshal([]byte(m[3]), &patterns), m[1])
		active = append(active, Brand{Name: m[2], Query: m[4], Patterns: patterns})
	}
	assert.Equal(t, DefaultBrands, active)
}
