package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPlacesKey = "TEST-PLACES-KEY-not-a-real-key"

// placesCall is one request the fake Google saw.
type placesCall struct {
	endpoint string
	mask     string
	key      string
	url      string
	body     map[string]interface{}
}

// fakeGoogle answers from testdata fixtures (no live call): text search by
// textQuery, nearby search by its first included type; override can
// replace an answer (status, fixture file or raw body).
type fakeGoogle struct {
	t        *testing.T
	mu       sync.Mutex
	calls    []placesCall
	override func(c placesCall) (status int, body string, ok bool)
}

func fixture(t *testing.T, name string) string {
	raw, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	return string(raw)
}

func (f *fakeGoogle) RoundTrip(r *http.Request) (*http.Response, error) {
	raw, _ := io.ReadAll(r.Body)
	c := placesCall{endpoint: r.URL.Path[strings.LastIndex(r.URL.Path, ":")+1:], mask: r.Header.Get("X-Goog-FieldMask"), key: r.Header.Get("X-Goog-Api-Key"), url: r.URL.String()}
	require.NoError(f.t, json.Unmarshal(raw, &c.body))
	f.mu.Lock()
	f.calls = append(f.calls, c)
	f.mu.Unlock()

	if f.override != nil {
		if status, body, ok := f.override(c); ok {
			return answer(status, body)
		}
	}
	name := "{}"
	switch c.endpoint {
	case "searchText":
		switch c.body["textQuery"] {
		case "Indomaret":
			name = fixture(f.t, "places_text_indomaret.json")
		case "Alfamart":
			name = fixture(f.t, "places_text_alfamart.json")
		}
	case "searchNearby":
		switch c.body["includedTypes"].([]interface{})[0] {
		case "convenience_store":
			name = fixture(f.t, "places_nearby_convenience.json")
		case "corporate_office":
			name = fixture(f.t, "places_nearby_office.json")
		}
	}
	return answer(200, name)
}

func (f *fakeGoogle) endpoints() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, c := range f.calls {
		out = append(out, c.endpoint)
	}
	return out
}

func placesClient(t *testing.T, cats ...string) (*PlacesClient, *fakeGoogle) {
	if len(cats) == 0 {
		cats = DefaultNearbyCategories
	}
	f := &fakeGoogle{t: t}
	return &PlacesClient{apiKey: testPlacesKey, baseURL: "https://places.example.test/v1/places", http: &http.Client{Transport: f}, categories: cats}, f
}

func names(places []dto.NearbyPlaceItem) []string {
	var out []string
	for _, p := range places {
		out = append(out, p.Category+"|"+p.Brand+"|"+p.Name)
	}
	sort.Strings(out)
	return out
}

func TestPlacesFetchArea_DefaultCategories(t *testing.T) {
	c, g := placesClient(t)
	taken := 0

	area, err := c.FetchArea(context.Background(), dagoLat, dagoLng, 620, func(context.Context) error { taken++; return nil })

	require.NoError(t, err)
	assert.Equal(t, 7, c.CallsPerArea())
	assert.Equal(t, 7, area.Calls)
	assert.Equal(t, 7, taken, "every call goes through the daily cap")
	require.Len(t, g.calls, 7)

	// office first and alone (a refused key wastes 1 call)
	first := g.calls[0]
	assert.Equal(t, "searchNearby", first.endpoint)
	assert.Equal(t, []interface{}{"corporate_office"}, first.body["includedTypes"])

	for _, call := range g.calls {
		assert.Equal(t, testPlacesKey, call.key, "key in the header")
		assert.NotContains(t, call.url, testPlacesKey, "key never in the URL")
		assert.Equal(t, "id", call.body["languageCode"])
		assert.Equal(t, "ID", call.body["regionCode"])
		assert.Equal(t, "DISTANCE", call.body["rankPreference"])
		if call.endpoint == "searchText" {
			assert.Equal(t, "places.id,places.displayName,places.location,places.types,places.primaryType,nextPageToken", call.mask)
			assert.Contains(t, call.body, "locationBias", "brand search: bias circle, not a rectangle")
			assert.EqualValues(t, 20, call.body["pageSize"])
		} else {
			assert.Equal(t, "places.id,places.displayName,places.location,places.types,places.primaryType", call.mask)
			assert.Contains(t, call.body, "locationRestriction")
			assert.EqualValues(t, 20, call.body["maxResultCount"])
		}
	}
	var queries []string
	for _, call := range g.calls {
		if q, ok := call.body["textQuery"].(string); ok {
			queries = append(queries, q)
		}
	}
	sort.Strings(queries)
	assert.Equal(t, []string{"Alfamart", "Alfamidi", "Circle K", "Indomaret", "Lawson"}, queries)

	var competitors []dto.NearbyPlaceItem
	for _, p := range area.Places {
		if p.Category == constants.NearbyCategoryCompetitor {
			competitors = append(competitors, p)
		}
		assert.Equal(t, constants.NearbySourceGoogle, p.Source)
		assert.Equal(t, "g:"+p.PlaceID, p.ID)
		assert.NotContains(t, p.Name, "FamilyMart")
	}
	// ATM, warehouse, FamilyMart, other brand, outside the circle: dropped;
	// "Indomart" / "Alfa Expres" spellings kept; the Indomaret found twice
	// (brand search + convenience_store) counts once
	assert.Equal(t, []string{
		"competitor|7-Eleven|7-Eleven Dago",
		"competitor|Alfamart|Alfa Expres Dago",
		"competitor|Alfamart|Alfamart Dipatiukur",
		"competitor|Indomaret|Indomaret Dago Atas",
		"competitor|Indomaret|Indomart Simpang",
		"competitor|Lainnya|Toko Sumber Rejeki",
	}, names(competitors))

	assert.Equal(t, []string{"competitor", "office"}, keys(area.Categories))
	assert.Equal(t, dto.NearbyCategoryInfo{Count: 6, Source: "google"}, area.Categories["competitor"], "Indomaret: next page offered but the farthest answer is outside, so complete")
	office := area.Categories["office"]
	assert.Equal(t, 19, office.Count)
	assert.True(t, office.Capped, "20 answers = Google's limit")
	require.NotNil(t, office.CompleteWithinM)
	assert.Equal(t, 422, *office.CompleteWithinM)
	assert.Empty(t, area.Failed)
}

func keys(m map[string]dto.NearbyCategoryInfo) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestPlacesFetchArea_DeniedOnFirstCall(t *testing.T) {
	c, g := placesClient(t)
	g.override = func(placesCall) (int, string, bool) {
		return 403, fixture(t, "places_denied_service_disabled.json"), true
	}

	area, err := c.FetchArea(context.Background(), dagoLat, dagoLng, 620, nil)

	var denied *PlacesDeniedError
	require.True(t, errors.As(err, &denied))
	assert.Equal(t, "SERVICE_DISABLED", denied.Code)
	assert.Equal(t, "Places API (New) belum diaktifkan di project Google Cloud kunci ini", denied.Reason)
	assert.Equal(t, 1, area.Calls, "stops after the first call")
	assert.Len(t, g.calls, 1)
	assert.NotContains(t, err.Error(), testPlacesKey)
}

func TestPlacesFetchArea_BlockedKeyLaterIsFatal(t *testing.T) {
	c, g := placesClient(t)
	g.override = func(call placesCall) (int, string, bool) {
		if call.body["textQuery"] == "Lawson" {
			return 403, `{"error":{"status":"PERMISSION_DENIED","details":[{"reason":"API_KEY_SERVICE_BLOCKED"}]}}`, true
		}
		return 0, "", false
	}

	_, err := c.FetchArea(context.Background(), dagoLat, dagoLng, 620, nil)

	var denied *PlacesDeniedError
	require.True(t, errors.As(err, &denied))
	assert.Equal(t, "Kunci Google belum diizinkan untuk Places API (New)", denied.Reason)
}

func TestPlacesFetchArea_FailedCategoryAndFirstFailure(t *testing.T) {
	c, g := placesClient(t, constants.NearbyCategoryCompetitor, constants.NearbyCategoryOffice, constants.NearbyCategorySchool)
	assert.Equal(t, 8, c.CallsPerArea())
	g.override = func(call placesCall) (int, string, bool) {
		if types, ok := call.body["includedTypes"].([]interface{}); ok && types[0] == "school" {
			return 500, `{"error":{"status":"INTERNAL"}}`, true
		}
		return 0, "", false
	}

	area, err := c.FetchArea(context.Background(), dagoLat, dagoLng, 620, nil)

	require.NoError(t, err)
	assert.Equal(t, []string{"school"}, area.Failed, "the service fills it from OSM")
	assert.Equal(t, []string{"competitor", "office"}, keys(area.Categories))

	// the first (office) call failing fails the whole area
	g.override = func(call placesCall) (int, string, bool) { return 503, `{"error":{"status":"UNAVAILABLE"}}`, true }
	_, err = c.FetchArea(context.Background(), dagoLat, dagoLng, 620, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 503 UNAVAILABLE")
}

func TestPlacesFetchArea_DailyCapStops(t *testing.T) {
	c, g := placesClient(t)
	n := 0

	area, err := c.FetchArea(context.Background(), dagoLat, dagoLng, 620, func(context.Context) error {
		n++
		if n > 3 {
			return ErrPlacesDailyCap
		}
		return nil
	})

	assert.ErrorIs(t, err, ErrPlacesDailyCap)
	assert.LessOrEqual(t, area.Calls, 3)
	assert.Equal(t, area.Calls, len(g.endpoints()), "no call past the cap")
}

func TestPlacesClient_NotConfiguredAndCategories(t *testing.T) {
	p := NewPlacesClient(nil)
	assert.False(t, p.Configured())
	assert.Equal(t, 7, p.CallsPerArea())
	_, err := p.FetchArea(context.Background(), 0, 0, 0, nil)
	assert.ErrorIs(t, err, ErrPlacesNotSetUp)

	all, _ := placesClient(t, constants.NearbyCategories...)
	assert.Equal(t, 10, all.CallsPerArea(), "all five categories = the old 10 calls")
	assert.Equal(t, DefaultNearbyCategories, NearbyCategoriesFrom(nil))
}
