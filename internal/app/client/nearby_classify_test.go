package client

import (
	"os"
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
		"Circle-K": "Circle K", "7-Eleven": "7-Eleven", "K3Mart": "K3 Mart", "Warung Madura": "", "Circlekebab Corner": "",
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

	places, err := ParseOverpassArea(raw, dagoLat, dagoLng, 620)

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

	_, err = ParseOverpassArea([]byte("<html>busy</html>"), dagoLat, dagoLng, 500)
	assert.Error(t, err)
	_, err = ParseOverpassArea([]byte(`{"remark":"runtime error"}`), dagoLat, dagoLng, 500)
	assert.Error(t, err)
}

func TestOverpassAreaQuery(t *testing.T) {
	q := OverpassAreaQuery(dagoLat, dagoLng, 500)
	assert.True(t, strings.HasPrefix(q, "[out:json][timeout:60][bbox:-6.89049,107.60848,-6.88151,107.61752];"), q[:80])
	assert.True(t, strings.HasSuffix(q, "out center tags 5000;"))
}
