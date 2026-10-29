package router

// Check of the rewritten local §6 (site-dev-api local commit 2191e07):
// GET /v1/site/nearby-places and the map data in the AI insight. Google
// Places and OpenStreetMap are mocks (real clients are TODOs, plan Q4).

import (
	"context"
	"database/sql/driver"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"site-dev-backend/internal/app/client"
	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- copies of site-dev-api 2191e07 internal/app/dto/nearby_place.go ------

type oldNearbyPlaceResp struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Category  string  `json:"category"`
	Brand     string  `json:"brand,omitempty"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	DistanceM int     `json:"distance_m"`
	Source    string  `json:"source"`
	PlaceID   string  `json:"place_id,omitempty"`
}

type oldNearbyCategoryInfo struct {
	Count           int    `json:"count"`
	Source          string `json:"source"`
	Capped          bool   `json:"capped"`
	CompleteWithinM *int   `json:"complete_within_m,omitempty"`
}

type oldNearestStoreResp struct {
	Name      string `json:"name"`
	Code      string `json:"code"`
	DistanceM int    `json:"distance_m"`
	Within1Km int    `json:"within_1km"`
}

type oldNearbyAreaResp struct {
	Source            string                           `json:"source"`
	Lat               float64                          `json:"lat"`
	Lng               float64                          `json:"lng"`
	RadiusM           int                              `json:"radius_m"`
	FetchedAt         time.Time                        `json:"fetched_at"`
	Places            []oldNearbyPlaceResp             `json:"places"`
	Categories        map[string]oldNearbyCategoryInfo `json:"categories"`
	GoogleCalls       int                              `json:"google_calls"`
	Cached            bool                             `json:"cached"`
	FallbackReason    string                           `json:"fallback_reason,omitempty"`
	NearestFamilyMart *oldNearestStoreResp             `json:"nearest_familymart"`
	StoreSource       string                           `json:"store_source"`
	Attribution       string                           `json:"attribution"`
}

// ---------------------------------------------------------------------------

const dLat, dLng = -6.8860, 107.6130

func placesOn(t *testing.T) *mocks.IOverpassClient {
	testEnv.Set("places.enabled", true)
	// all five categories = the 2191e07 answer these tests compare with
	// (the default since 22 Oct is competitor + office, API Lab demo view)
	testEnv.Set("places.categories", constants.NearbyCategories)
	t.Cleanup(func() {
		testEnv.Set("places.enabled", false)
		testEnv.Set("places.categories", []string{})
	})
	osm := mocks.NewIOverpassClient(t)
	withClients(t, func(c *client.Clients) { c.Overpass = osm })
	raw, err := os.ReadFile("../client/testdata/overpass_dago.json")
	require.NoError(t, err)
	osm.On("FetchArea", mock.Anything, -6.886, 107.613, 620, mock.Anything).Return(func(_ context.Context, lat, lng float64, r int, brands client.BrandSet) ([]dto.NearbyPlaceItem, error) {
		return client.ParseOverpassArea(raw, lat, lng, r, brands)
	})
	return osm
}

var storePoints = sqlfake.Result{Match: `SELECT
		ms.StoreCode,
		ms.StoreName,
		ms.Longitude,
		ms.Latitude
	FROM AMFIBI_MASTER.dbo.MStore as ms
	WHERE`, Columns: []string{"StoreCode", "StoreName", "Longitude", "Latitude"}, Rows: [][]driver.Value{{"3001", "Dago Atas", dLng, dLat - 0.004}}}

func TestNearbyPlaces(t *testing.T) {
	osm := placesOn(t)
	redis := withRedis(t)
	h, _, _ := stack(t, storePoints)

	code, body := call(t, h, http.MethodGet, "/v1/site/nearby-places?lat=-6.886&lng=107.613", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	// rebuild the expected answer with the 2191e07 types from our own JSON,
	// so a renamed or missing field fails
	var want oldNearbyAreaResp
	require.NoError(t, jsonUnmarshalData(body, &want))
	assertSameJSON(t, oldDataEnvelope(t, want), body)
	assert.Equal(t, "osm", want.Source)
	assert.Equal(t, 500, want.RadiusM)
	assert.Equal(t, "real", want.StoreSource)
	assert.Equal(t, "Kunci Google Places belum diisi di server ini", want.FallbackReason)
	assert.Equal(t, &oldNearestStoreResp{Name: "Dago Atas", Code: "3001", DistanceM: 445, Within1Km: 1}, want.NearestFamilyMart)
	assert.Equal(t, 3, want.Categories["competitor"].Count)
	assert.Equal(t, "Indomaret Fresh Dago", want.Places[0].Name)
	assert.Equal(t, 6*time.Hour, redis.TTL("site-dev-api:test:cache:nearby_area:v3:competitor+office+school+university+apartment:b"+client.DefaultBrandSet().Key()+":-6.886,107.613:500:"), "OSM fallback cached 6 h")

	code, body = call(t, h, http.MethodGet, "/v1/site/nearby-places?lat=-6.886&lng=107.613", nil, 1001)
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, string(body), `"cached":true`)
	osm.AssertNumberOfCalls(t, "FetchArea", 1)
}

func TestNearbyPlaces_OffByDefault(t *testing.T) {
	h, _, _ := stack(t)

	code, body := call(t, h, http.MethodGet, "/v1/site/nearby-places?lat=-6.886&lng=107.613", nil, 1001)

	assert.Equal(t, http.StatusServiceUnavailable, code)
	assertSameJSON(t, []byte(`{"code":503,"status":"failure","message":"Service Unavailable","errors":{"error":"Data sekitar titik belum diaktifkan di server ini ([places] enabled = false)."}}`), body)
}

func TestAIInsight_WithMapData(t *testing.T) {
	placesOn(t)
	gemini := withGemini(t)
	withRedis(t)
	gemini.On("GenerateInsight", mock.Anything, mock.MatchedBy(func(in client.GeminiInput) bool {
		return in.Measured != nil && in.Measured.Source == "osm" && in.Measured.Counts["office"] == 2
	})).Return(geminiInsight(), "gemini-2.5-flash", nil)
	site := siteST1Row()
	site[10], site[11] = dLat, dLng
	h, _, _ := stack(t, append([]sqlfake.Result{storePoints, {Match: `INSERT INTO "SiteAIInsights"`, Columns: []string{"SiteAIID"}, Rows: [][]driver.Value{{int64(77)}}},
		{Match: `FROM "Sites" WHERE SiteID = @p1`, Columns: siteColumns, Rows: [][]driver.Value{site}}}, detailResults("USER")...)...)

	code, body := call(t, h, http.MethodPost, "/v1/task/ai-insight", map[string]any{"site_id": "ST1"}, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Contains(t, string(body), `"data_source":"gemini+osm","measured":{"source":"osm","radius_m":500,"counts":{"apartment":1,"competitor":3,"office":2,"school":2,"university":1}`)
	assert.True(t, strings.Contains(string(body), `"nearest_familymart":{"name":"Dago Atas","code":"3001","distance_m":445,"within_1km":1}`))
}

// Wes, 29 Oct: ?include=other_store adds the convenience stores of no active
// brand (category other_store), from the same cached search; off by default
// ([places] other_stores) and never without the include.
func TestNearbyPlaces_OtherStores(t *testing.T) {
	osm := placesOn(t)
	withRedis(t)
	h, _, _ := stack(t, storePoints)

	// switch off: the include is ignored, the answer is the old one
	code, plain := call(t, h, http.MethodGet, "/v1/site/nearby-places?lat=-6.886&lng=107.613&include=other_store", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(plain))
	assert.NotContains(t, string(plain), "other_store")

	testEnv.Set("places.other_stores", true)
	t.Cleanup(func() { testEnv.Set("places.other_stores", false) })
	h, _, _ = stack(t, storePoints, storePoints)

	// on, but not asked: unchanged
	code, body := call(t, h, http.MethodGet, "/v1/site/nearby-places?lat=-6.886&lng=107.613", nil, 1001)
	require.Equal(t, http.StatusOK, code)
	assert.NotContains(t, string(body), "other_store")

	code, body = call(t, h, http.MethodGet, "/v1/site/nearby-places?lat=-6.886&lng=107.613&include=other_store", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	var got oldNearbyAreaResp
	require.NoError(t, jsonUnmarshalData(body, &got))
	other := got.Categories[constants.NearbyCategoryOtherStore]
	assert.Equal(t, "osm", other.Source)
	assert.Equal(t, 3, got.Categories["competitor"].Count, "listed brands unchanged")
	brands := client.DefaultBrandSet()
	n, last := 0, -1
	for _, p := range got.Places {
		assert.GreaterOrEqual(t, p.DistanceM, last, "nearest first over all groups")
		last = p.DistanceM
		if p.Category != constants.NearbyCategoryOtherStore {
			continue
		}
		n++
		assert.Empty(t, p.Brand)
		assert.NotEmpty(t, p.Name)
		assert.Empty(t, brands.Of(p.Name), "a listed brand is never an other store: %s", p.Name)
		assert.LessOrEqual(t, p.DistanceM, 500)
		assert.NotContains(t, strings.ToLower(p.Name), "familymart")
		assert.NotContains(t, strings.ToLower(p.Name), "vapor")
	}
	assert.Equal(t, other.Count, n)
	assert.Greater(t, n, 0, "the Dago sample has unbranded minimarkets")
	assert.Contains(t, string(body), `"name":"Warung Bu Ani","category":"other_store"`)
	assert.Contains(t, string(body), `"address":"Jalan Dipatiukur No. 7, Lebakgede"`, "OSM addr:* tags → address (Wes, 29 Oct)")
	osm.AssertNumberOfCalls(t, "FetchArea", 1)
}
