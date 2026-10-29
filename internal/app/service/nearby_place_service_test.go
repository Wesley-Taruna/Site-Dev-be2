package service

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"site-dev-backend/internal/app/client"
	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/app/model"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const dagoLat, dagoLng = -6.8860, 107.6130

func dagoPlaces(t *testing.T) []dto.NearbyPlaceItem {
	raw, err := os.ReadFile("../client/testdata/overpass_dago.json")
	require.NoError(t, err)
	places, err := client.ParseOverpassArea(raw, dagoLat, dagoLng, 620, client.DefaultBrandSet())
	require.NoError(t, err)
	return places
}

type nearbyMocks struct {
	area     *mocks.INearbyAreaRepository
	site     *mocks.ISiteRepository
	places   *mocks.IPlacesClient
	overpass *mocks.IOverpassClient
}

// allNearbyKey: the cache key prefix with all five categories on (the
// site-dev-api §6 behaviour these tests were written for).
var allNearbyKey = "v3:competitor+office+school+university+apartment:b" + client.DefaultBrandSet().Key() + ":"

func newTestNearby(t *testing.T, enabled bool) (*NearbyPlaceService, nearbyMocks) {
	m := nearbyMocks{mocks.NewINearbyAreaRepository(t), mocks.NewISiteRepository(t), mocks.NewIPlacesClient(t), mocks.NewIOverpassClient(t)}
	cfg := loadNearbyConfig(nil)
	cfg.enabled = enabled
	cfg.categories = constants.NearbyCategories
	return &NearbyPlaceService{opt: testOptions(), RepositoryNearbyArea: m.area, RepositorySite: m.site, ClientPlaces: m.places, ClientOverpass: m.overpass,
		cfg: cfg, state: &nearbyState{inflight: map[string]*inflightArea{}}, now: func() time.Time { return time.Date(2026, 10, 21, 8, 0, 0, 0, time.UTC) }}, m
}

func TestNearbyGetArea_OffIs503(t *testing.T) {
	s, _ := newTestNearby(t, false)

	_, err := s.GetArea(context.Background(), dto.NearbyPlacesRequestGet{Lat: dagoLat, Lng: dagoLng})

	var he *echo.HTTPError
	require.True(t, errors.As(err, &he))
	assert.Equal(t, http.StatusServiceUnavailable, he.Code)
}

func TestNearbyGetArea_OSMWhenNoGoogleKey(t *testing.T) {
	s, m := newTestNearby(t, true)
	m.area.On("GetArea", ctxAny, allNearbyKey+"-6.886,107.613:500").Return(nil, false).Once()
	m.places.On("Configured").Return(false)
	m.overpass.On("FetchArea", ctxAny, -6.886, 107.613, 620, mock.Anything).Return(dagoPlaces(t), nil).Once()
	m.area.On("SetArea", ctxAny, allNearbyKey+"-6.886,107.613:500", mock.Anything, 6*time.Hour).Return().Once()
	m.site.On("GetActiveStorePoints", ctxAny).Return([]model.StoreMap{{StoreCode: "3001", StoreName: "Dago Atas", Latitude: dagoLat - 0.004, Longitude: dagoLng}}, nil).Once()

	res, err := s.GetArea(context.Background(), dto.NearbyPlacesRequestGet{Lat: dagoLat, Lng: dagoLng})

	require.NoError(t, err)
	assert.Equal(t, constants.NearbySourceOSM, res.Source)
	assert.Equal(t, 500, res.RadiusM)
	assert.Equal(t, "Kunci Google Places belum diisi di server ini", res.FallbackReason)
	assert.Equal(t, "© OpenStreetMap contributors", res.Attribution)
	assert.False(t, res.Cached)
	got := map[string]int{}
	for c, info := range res.Categories {
		got[c] = info.Count
	}
	// same counts as site-dev-api 2191e07 TestAreaForPointCountsEqualList
	assert.Equal(t, map[string]int{"competitor": 3, "office": 2, "school": 2, "university": 1, "apartment": 1}, got)
	require.NotNil(t, res.NearestFamilyMart)
	assert.Equal(t, dto.NearestStoreResponse{Name: "Dago Atas", Code: "3001", DistanceM: 445, Within1Km: 1}, *res.NearestFamilyMart)

	// prompt data from the same area
	measured := measuredFromArea(res)
	in := client.GeminiInput{Site: &model.Site{SiteName1: "X", SiteCity: "Bandung"}, Measured: measured}
	body, err := client.GeminiRequest(in)
	require.NoError(t, err)
	for _, want := range []string{
		"DATA TERUKUR (OpenStreetMap, radius 500 m dari koordinat",
		"- Pesaing bermerek (minimarket dan kedai kopi): 3 tercatat di OpenStreetMap → Indomaret Fresh Dago 101 m; Circle K Dago 134 m; Alfamidi Dipatiukur 248 m",
		"- Kantor: 2 tempat bernama tercatat di OpenStreetMap (contoh terdekat: Otoritas Jasa Keuangan 135 m; Graha Dago 223 m)",
		"- FamilyMart terdekat (daftar toko FamilyMart): Dago Atas (kode 3001), 445 m; FamilyMart dalam radius 1 km: 1",
		"E) DATA TERUKUR (WAJIB DIPAKAI bila tersedia)",
	} {
		assert.Contains(t, strings.ReplaceAll(string(body), `>`, ">"), want)
	}
}

func TestNearbyGetArea_CachedAndRadiusLimit(t *testing.T) {
	s, m := newTestNearby(t, true)
	m.area.On("GetArea", ctxAny, allNearbyKey+"-6.886,107.613:300").Return([]byte(`{"at":1,"lat":-6.886,"lng":107.613,"radius":420,"source":"osm","places":[],"categories":{},"google_calls":0}`), true)
	m.site.On("GetActiveStorePoints", ctxAny).Return(nil, errors.New("no access to AMFIBI_MASTER"))

	res, err := s.GetArea(context.Background(), dto.NearbyPlacesRequestGet{Lat: dagoLat, Lng: dagoLng, RadiusM: 300})

	require.NoError(t, err)
	assert.True(t, res.Cached)
	assert.Nil(t, res.NearestFamilyMart, "store list failure is not an error")
	assert.Equal(t, []dto.NearbyPlaceItem{}, res.Places)

	_, err = s.GetArea(context.Background(), dto.NearbyPlacesRequestGet{Lat: dagoLat, Lng: dagoLng, RadiusM: 1001})
	assert.Contains(t, err.Error(), "radius maksimal 1000 m")
	_, err = s.GetArea(context.Background(), dto.NearbyPlacesRequestGet{Lat: 0, Lng: 0})
	assert.Contains(t, err.Error(), "Koordinat tidak valid")
}

func TestNearbyBuild_GoogleDeniedFallsBackAndSwitchesOff(t *testing.T) {
	s, m := newTestNearby(t, true)
	m.places.On("Configured").Return(true)
	m.places.On("CallsPerArea", mock.Anything).Return(10)
	m.area.On("GoogleCallsLeft", ctxAny, int64(150)).Return(int64(150))
	m.places.On("FetchArea", ctxAny, dagoLat, dagoLng, 620, mock.Anything, mock.Anything).Return(client.PlacesArea{Calls: 1}, &client.PlacesDeniedError{Reason: "Places API (New) belum diaktifkan di project Google Cloud kunci ini"}).Once()
	m.overpass.On("FetchArea", ctxAny, dagoLat, dagoLng, 620, mock.Anything).Return(dagoPlaces(t), nil)

	st, err := s.build(context.Background(), dagoLat, dagoLng, 620, client.DefaultBrandSet())

	require.NoError(t, err)
	assert.Equal(t, constants.NearbySourceOSM, st.Source)
	assert.Equal(t, 1, st.GoogleCalls)
	assert.Equal(t, "Places API (New) belum diaktifkan di project Google Cloud kunci ini", st.FallbackReason)

	// an hour off: Google not asked again
	st, err = s.build(context.Background(), dagoLat, dagoLng, 620, client.DefaultBrandSet())
	require.NoError(t, err)
	assert.Equal(t, "Places API (New) belum diaktifkan di project Google Cloud kunci ini", st.FallbackReason)
	m.places.AssertNumberOfCalls(t, "FetchArea", 1)
}

func TestNearbyBuild_GoogleWithFailedCategoryIsMixed(t *testing.T) {
	s, m := newTestNearby(t, true)
	m.places.On("Configured").Return(true)
	m.places.On("CallsPerArea", mock.Anything).Return(10)
	m.area.On("GoogleCallsLeft", ctxAny, int64(150)).Return(int64(40))
	m.area.On("TakeGoogleCall", ctxAny, int64(150)).Return(nil)
	m.places.On("FetchArea", ctxAny, dagoLat, dagoLng, 620, mock.Anything, mock.Anything).Return(func(ctx context.Context, _ float64, _ float64, _ int, _ client.BrandSet, take func(context.Context) error) (client.PlacesArea, error) {
		require.NoError(t, take(ctx))
		return client.PlacesArea{
			Places:     []dto.NearbyPlaceItem{{ID: "g:1", Name: "Indomaret Dago", Category: "competitor", Brand: "Indomaret", Lat: dagoLat, Lng: dagoLng, Source: "google"}},
			Categories: map[string]dto.NearbyCategoryInfo{"competitor": {Count: 1, Source: "google"}},
			Failed:     []string{"school"}, Calls: 10,
		}, nil
	})
	m.overpass.On("FetchArea", ctxAny, dagoLat, dagoLng, 620, mock.Anything).Return(dagoPlaces(t), nil)

	st, err := s.build(context.Background(), dagoLat, dagoLng, 620, client.DefaultBrandSet())

	require.NoError(t, err)
	assert.Equal(t, constants.NearbySourceMixed, st.Source)
	assert.Equal(t, 10, st.GoogleCalls)
	assert.Equal(t, dto.NearbyCategoryInfo{Count: 3, Source: "osm"}, st.Categories["school"])
}

func TestNearbyGoogleUsable_DailyCap(t *testing.T) {
	s, m := newTestNearby(t, true)
	m.places.On("Configured").Return(true)
	m.places.On("CallsPerArea", mock.Anything).Return(9)
	m.area.On("GoogleCallsLeft", ctxAny, int64(150)).Return(int64(8)).Once()

	ok, reason := s.googleUsable(context.Background(), client.DefaultBrandSet())

	assert.False(t, ok)
	assert.Equal(t, "Batas harian panggilan Google hampir habis", reason)

	m.area.On("GoogleCallsLeft", ctxAny, int64(150)).Return(int64(9)).Once()
	ok, _ = s.googleUsable(context.Background(), client.DefaultBrandSet())
	assert.True(t, ok, "9 left is enough for one area of 9 calls (23 Oct: 7 brand searches + convenience_store + office)")
}

// Default categories (API Lab demo view): only competitors and offices are
// answered and given to the prompt, even when OSM has more.
func TestNearbyGetArea_DefaultCategoriesOnly(t *testing.T) {
	s, m := newTestNearby(t, true)
	s.cfg.categories = loadNearbyConfig(nil).categories
	key := "v3:competitor+office:b" + client.DefaultBrandSet().Key() + ":-6.886,107.613:500"
	m.area.On("GetArea", ctxAny, key).Return(nil, false).Once()
	m.places.On("Configured").Return(false)
	m.overpass.On("FetchArea", ctxAny, -6.886, 107.613, 620, mock.Anything).Return(dagoPlaces(t), nil).Once()
	m.area.On("SetArea", ctxAny, key, mock.Anything, 6*time.Hour).Return().Once()
	m.site.On("GetActiveStorePoints", ctxAny).Return(nil, nil).Once()

	res, err := s.GetArea(context.Background(), dto.NearbyPlacesRequestGet{Lat: dagoLat, Lng: dagoLng})

	require.NoError(t, err)
	got := map[string]int{}
	for c, info := range res.Categories {
		got[c] = info.Count
	}
	assert.Equal(t, map[string]int{"competitor": 3, "office": 2}, got)
	for _, p := range res.Places {
		assert.Contains(t, []string{"competitor", "office"}, p.Category)
	}

	measured := measuredFromArea(res)
	assert.Equal(t, map[string]int{"competitor": 3, "office": 2}, measured.Counts)
	body, err := client.GeminiRequest(client.GeminiInput{Site: &model.Site{SiteName1: "X", SiteCity: "Bandung"}, Measured: measured})
	require.NoError(t, err)
	assert.Contains(t, string(body), "- Kantor: 2 tempat bernama")
	assert.NotContains(t, string(body), "- Sekolah", "switched-off groups are not given to the AI as 0")
}

func TestMeasuredOffIsNil(t *testing.T) {
	s, _ := newTestNearby(t, false)
	m, err := s.Measured(context.Background(), dagoLat, dagoLng)
	assert.NoError(t, err)
	assert.Nil(t, m)
}

func TestAIInsight_DataSourceOnlyWhenPlacesOn(t *testing.T) {
	nearby := mocks.NewINearbyPlaceService(t)
	s := &AIInsightService{opt: testOptions(), ServicesNearby: nearby}
	nearby.On("Enabled").Return(false).Once()
	insight := dto.AIInsightResponseGet{}
	s.markDataSource(&insight, nil)
	assert.Equal(t, "", insight.DataSource)

	nearby.On("Enabled").Return(true)
	s.markDataSource(&insight, &dto.NearbyMeasured{Source: "osm"})
	assert.Equal(t, "gemini+osm", insight.DataSource)
	s.markDataSource(&insight, nil)
	assert.Equal(t, "gemini", insight.DataSource)

	nearby.On("Measured", ctxAny, dagoLat, dagoLng).Return(nil, errors.New("overpass down"))
	assert.Nil(t, s.measuredFor(context.Background(), dagoLat, dagoLng), "best effort")
}

// Brands come from the ACTIVE CompetitorBrands rows (Wes, 23 Oct), kept
// 10 minutes; unreadable table or no active row with a query = the code
// list.
func TestNearbyBrandSet_FromActiveRows(t *testing.T) {
	s, _ := newTestNearby(t, true)
	repo := mocks.NewICompetitorBrandRepository(t)
	s.RepositoryBrand = repo
	repo.On("Get", ctxAny).Return([]*model.CompetitorBrand{
		{Slug: "indomaret", Name: "Indomaret", Patterns: `["indo\\s*-?\\s*mare?t"]`, SearchQuery: "Indomaret", Active: true},
		{Slug: "k3-mart", Name: "K3 Mart", Patterns: `["k3\\s*mart"]`, Active: true},
		{Slug: "yomart", Name: "Yomart", Patterns: `["yo\\s*mart"]`, SearchQuery: "Yomart", Active: false},
		{Slug: "tomoro", Name: "Tomoro Coffee", Patterns: `not json`, SearchQuery: "Tomoro Coffee", Active: true},
	}, nil).Once()

	set := s.brandSet(context.Background())

	assert.Equal(t, []string{"Indomaret", "K3 Mart", "Tomoro Coffee"}, set.Names(), "inactive Yomart left out")
	assert.Len(t, set.Searched(), 2, "K3 Mart has no own search")
	assert.Equal(t, "Tomoro Coffee", set.Of("TOMORO COFFEE Kuningan"), "broken patterns: the name is used")
	assert.Equal(t, "", set.Of("Yomart Tebet"))
	assert.Equal(t, set.Key(), s.brandSet(context.Background()).Key(), "kept in memory (Get once)")

	// after 10 minutes: asked again; an error = the code list
	s.now = func() time.Time { return time.Date(2026, 10, 21, 8, 11, 0, 0, time.UTC) }
	repo.On("Get", ctxAny).Return(nil, errors.New("Invalid object name 'CompetitorBrands'")).Once()
	assert.Equal(t, client.DefaultBrandSet().Names(), s.brandSet(context.Background()).Names())

	// rows without SearchQuery (old table) = the code list too
	s.now = func() time.Time { return time.Date(2026, 10, 21, 8, 22, 0, 0, time.UTC) }
	repo.On("Get", ctxAny).Return([]*model.CompetitorBrand{{Slug: "indomaret", Name: "Indomaret", Active: true}}, nil).Once()
	assert.Equal(t, client.DefaultBrandSet().Names(), s.brandSet(context.Background()).Names())
}

// Wes, 29 Oct: other_store = competitor-search places of no active brand.
func TestWithOtherStores(t *testing.T) {
	within := 400
	st := &storedArea{Source: constants.NearbySourceGoogle, Lat: dagoLat, Lng: dagoLng, Places: []dto.NearbyPlaceItem{
		{ID: "g:1", Name: "Indomaret Dago", Category: "competitor", Brand: "Indomaret", Lat: dagoLat + 0.001, Lng: dagoLng},
		{ID: "g:2", Name: "Toko Maju Jaya", Category: "competitor", Brand: client.BrandOther, Lat: dagoLat + 0.002, Lng: dagoLng},
		{ID: "g:3", Name: "", Category: "competitor", Brand: client.BrandOther, Lat: dagoLat, Lng: dagoLng},
		{ID: "g:4", Name: "Mart Jauh", Category: "competitor", Brand: client.BrandOther, Lat: dagoLat + 0.02, Lng: dagoLng},
		{ID: "g:5", Name: "Graha Dago", Category: "office", Lat: dagoLat, Lng: dagoLng + 0.001},
	}, Categories: map[string]dto.NearbyCategoryInfo{"competitor": {Source: constants.NearbySourceGoogle, Capped: true, CompleteWithinM: &within}}}
	places, cats := areaForPoint(st, dagoLat, dagoLng, 500, []string{"competitor", "office"})
	require.Len(t, places, 2)

	places, cats = withOtherStores(st, dagoLat, dagoLng, 500, places, cats)

	require.Len(t, places, 3)
	other := cats[constants.NearbyCategoryOtherStore]
	assert.Equal(t, 1, other.Count)
	assert.Equal(t, constants.NearbySourceGoogle, other.Source)
	assert.True(t, other.Capped, "same searches as competitor: capped follows")
	assert.Equal(t, 1, cats["competitor"].Count, "listed brands unchanged")
	var names []string
	for _, p := range places {
		names = append(names, p.Name+"/"+p.Category+"/"+p.Brand)
	}
	assert.Equal(t, []string{"Graha Dago/office/", "Indomaret Dago/competitor/Indomaret", "Toko Maju Jaya/other_store/"}, names)
}
