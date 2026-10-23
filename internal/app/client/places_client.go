package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/pkg/geo"

	"gitlab.com/fami-app-v2/familib/familabs/env"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var (
	// ErrPlacesNotSetUp: no Google key in the config (places.google_apikey).
	ErrPlacesNotSetUp = errors.New("places: Google Places key is not set (places.google_apikey)")
	// ErrPlacesDailyCap: the server-wide daily cap of Google calls is reached.
	ErrPlacesDailyCap = errors.New("places: daily Google call cap reached")
)

// PlacesDeniedError: Google refused the key for Places (API disabled, key
// restriction); Reason is plain Indonesian for the answer, Code is
// Google's reason (e.g. SERVICE_DISABLED).
type PlacesDeniedError struct {
	Reason string
	Code   string
}

func (e *PlacesDeniedError) Error() string { return "google places denied: " + e.Reason }

// PlacesArea is everything Google returned for one area: places, info per
// category, the categories whose calls failed (filled from OSM by the
// service), and how many calls were made.
type PlacesArea struct {
	Places     []dto.NearbyPlaceItem
	Categories map[string]dto.NearbyCategoryInfo
	Failed     []string
	Calls      int
}

// IPlacesClient is Google Places (New), the preferred source of nearby
// places.
type IPlacesClient interface {
	// Configured: a key is set (places.google_apikey).
	Configured() bool
	// CallsPerArea is how many Google calls one new area needs with the
	// categories switched on (places.categories) and these brands (one
	// Text Search per brand with a query).
	CallsPerArea(brands BrandSet) int
	// FetchArea fetches every category within radiusM; takeCall is called
	// before each Google call (daily cap) and its error stops the fetch.
	FetchArea(ctx context.Context, lat float64, lng float64, radiusM int, brands BrandSet, takeCall func(ctx context.Context) error) (area PlacesArea, err error)
}

// DefaultNearbyCategories are the groups searched when places.categories
// is not set: the API Lab demo view (Wes, 22 Oct): competitors and offices;
// schools, campuses and apartments off (no calls).
var DefaultNearbyCategories = []string{constants.NearbyCategoryCompetitor, constants.NearbyCategoryOffice}

// NearbyCategoriesFrom is places.categories in the fixed order
// (constants.NearbyCategories); unknown names are ignored, empty or unset
// = DefaultNearbyCategories. The Places client and the nearby service read
// the same key.
func NearbyCategoriesFrom(env env.Provider) []string {
	var want []string
	if env != nil && env.IsSet("places.categories") {
		want = env.GetStringSlice("places.categories")
	}
	on := map[string]bool{}
	for _, c := range want {
		on[strings.ToLower(strings.TrimSpace(c))] = true
	}
	out := make([]string, 0, len(constants.NearbyCategories))
	for _, c := range constants.NearbyCategories {
		if on[c] {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return append([]string(nil), DefaultNearbyCategories...)
	}
	return out
}

const (
	placesBaseURL = "https://places.googleapis.com/v1/places"
	// placesFieldMask: the cheapest mask that is enough (Text / Nearby
	// Search Pro); rating, phone, hours, website would make it Enterprise.
	placesFieldMask = "places.id,places.displayName,places.location,places.types,places.primaryType"
	// placesMaxResults: Google's maximum per Nearby Search and per Text
	// Search page; a full answer = the category may be capped.
	placesMaxResults = 20
	// placesMaxBody: answers are a few KB; anything bigger is cut.
	placesMaxBody = 5 << 20
)

// placesNearbyTypes: one Nearby Search per non-competitor category.
var placesNearbyTypes = map[string][]string{
	constants.NearbyCategoryOffice:     {"corporate_office"},
	constants.NearbyCategorySchool:     {"school", "primary_school", "secondary_school", "preschool"},
	constants.NearbyCategoryUniversity: {"university"},
	constants.NearbyCategoryApartment:  {"apartment_building", "apartment_complex", "condominium_complex"},
}

// PlacesClient is Google Places API (New), standard library only. The key
// goes in the X-Goog-Api-Key header, never in a URL, a log or an error.
type PlacesClient struct {
	apiKey     string
	baseURL    string
	http       *http.Client
	categories []string
}

// NewPlacesClient reads places.google_apikey (devops' secret store; empty
// = not configured, the service uses OpenStreetMap), places.categories and
// places.google_timeout_seconds (10).
func NewPlacesClient(env env.Provider) IPlacesClient {
	c := &PlacesClient{
		baseURL:    placesBaseURL,
		http:       &http.Client{Timeout: 10 * time.Second},
		categories: NearbyCategoriesFrom(env),
	}
	if env != nil {
		c.apiKey = strings.TrimSpace(env.GetString("places.google_apikey"))
		if n := env.GetInt("places.google_timeout_seconds"); n > 0 {
			c.http.Timeout = time.Duration(n) * time.Second
		}
	}
	return c
}

func (c *PlacesClient) Configured() bool { return c.apiKey != "" }

func (c *PlacesClient) CallsPerArea(brands BrandSet) int {
	n := 0
	for _, cat := range c.categories {
		n += len(c.searches(cat, 0, 0, 0, brands))
	}
	return n
}

// placeSearch is one Google call of a category.
type placeSearch struct {
	endpoint string // "searchText" or "searchNearby"
	body     map[string]interface{}
	// keep turns one Google place into an item (nil = drop)
	keep func(p dto.NearbyPlaceItem, primaryType string) *dto.NearbyPlaceItem
}

func placesCircle(lat float64, lng float64, radiusM int) map[string]interface{} {
	return map[string]interface{}{"circle": map[string]interface{}{
		"center": map[string]float64{"latitude": lat, "longitude": lng},
		"radius": radiusM,
	}}
}

// searches are the calls of one category (API Lab "Current code" + the
// sheet plan's coffee chains, Wes 23 Oct): competitors = one Text Search
// per brand with a query (7 by default: Indomaret, Alfamart, Alfamidi,
// Lawson, Circle K, Kopi Kenangan, Tomoro Coffee; locationBias circle +
// DISTANCE: the 20 nearest of the brand in distance order; a rectangle
// restriction gave relevance-trimmed lists) + 1 Nearby convenience_store
// (brands without a query, e.g. K3 Mart; other chains and unbranded kept
// as BrandOther, never counted); every other category = 1 Nearby Search.
func (c *PlacesClient) searches(cat string, lat float64, lng float64, radiusM int, brands BrandSet) []placeSearch {
	shop := func(p dto.NearbyPlaceItem, primaryType string) bool {
		return !IsNotAShop(p.Name) && !IsNotAShopType(primaryType) && !IsFamilyMart(p.Name)
	}
	nearby := func(types []string, keep func(p dto.NearbyPlaceItem, primaryType string) *dto.NearbyPlaceItem) placeSearch {
		return placeSearch{endpoint: "searchNearby", keep: keep, body: map[string]interface{}{
			"includedTypes":       types,
			"maxResultCount":      placesMaxResults,
			"rankPreference":      "DISTANCE",
			"locationRestriction": placesCircle(lat, lng, radiusM),
		}}
	}

	if cat != constants.NearbyCategoryCompetitor {
		types, ok := placesNearbyTypes[cat]
		if !ok {
			return nil
		}
		return []placeSearch{nearby(types, func(p dto.NearbyPlaceItem, _ string) *dto.NearbyPlaceItem {
			if IsFamilyMart(p.Name) {
				return nil
			}
			return &p
		})}
	}

	searched := brands.Searched()
	out := make([]placeSearch, 0, len(searched)+1)
	for _, brand := range searched {
		want := brand.Name
		out = append(out, placeSearch{endpoint: "searchText", body: map[string]interface{}{
			"textQuery":      brand.Query,
			"pageSize":       placesMaxResults,
			"rankPreference": "DISTANCE",
			"locationBias":   placesCircle(lat, lng, radiusM),
		}, keep: func(p dto.NearbyPlaceItem, primaryType string) *dto.NearbyPlaceItem {
			if brands.Of(p.Name) != want || !shop(p, primaryType) {
				return nil
			}
			p.Brand = want
			return &p
		}})
	}
	out = append(out, nearby([]string{"convenience_store"}, func(p dto.NearbyPlaceItem, primaryType string) *dto.NearbyPlaceItem {
		if !shop(p, primaryType) {
			return nil
		}
		p.Brand = brands.Of(p.Name)
		if p.Brand == "" {
			p.Brand = BrandOther
		}
		return &p
	}))
	return out
}

type googlePlace struct {
	ID          string `json:"id"`
	DisplayName struct {
		Text string `json:"text"`
	} `json:"displayName"`
	Location *struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
	Types       []string `json:"types"`
	PrimaryType string   `json:"primaryType"`
}

type googlePlacesBody struct {
	Places        []googlePlace `json:"places"`
	NextPageToken string        `json:"nextPageToken"`
}

type googlePlacesError struct {
	Error struct {
		Status  string `json:"status"`
		Details []struct {
			Reason string `json:"reason"`
		} `json:"details"`
	} `json:"error"`
}

// deniedReason is the Indonesian text for a refused key (same words as the
// prototype / API Lab).
func deniedReason(code string, status int) string {
	switch code {
	case "API_KEY_SERVICE_BLOCKED":
		return "Kunci Google belum diizinkan untuk Places API (New)"
	case "SERVICE_DISABLED":
		return "Places API (New) belum diaktifkan di project Google Cloud kunci ini"
	case "":
		return fmt.Sprintf("Google menolak Places API (%d)", status)
	}
	return "Google menolak Places API (" + code + ")"
}

var placesTracer = otel.Tracer("site-dev-backend/client")

// call sends one Places request. 401/403 = *PlacesDeniedError; the
// answer body is never quoted (only Google's status word).
func (c *PlacesClient) call(ctx context.Context, endpoint string, body map[string]interface{}) (res googlePlacesBody, err error) {
	ctx, span := placesTracer.Start(ctx, "PlacesClient."+endpoint)
	defer span.End()
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
	}()

	full := map[string]interface{}{"languageCode": "id", "regionCode": "ID"}
	for k, v := range body {
		full[k] = v
	}
	payload, err := json.Marshal(full)
	if err != nil {
		return res, fmt.Errorf("places: build request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+":"+endpoint, bytes.NewReader(payload))
	if err != nil {
		return res, fmt.Errorf("places: build request: %w", err)
	}
	mask := placesFieldMask
	if endpoint == "searchText" {
		mask += ",nextPageToken"
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", c.apiKey)
	req.Header.Set("X-Goog-FieldMask", mask)

	resp, err := c.http.Do(req)
	if err != nil {
		// same rule as the geocoding client: keep only the cause
		var uErr *url.Error
		if errors.As(err, &uErr) {
			err = uErr.Err
		}
		return res, fmt.Errorf("places %s: request failed: %w", endpoint, err)
	}
	defer resp.Body.Close()
	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	raw, err := io.ReadAll(io.LimitReader(resp.Body, placesMaxBody))
	if err != nil {
		return res, fmt.Errorf("places %s: read response: %w", endpoint, err)
	}
	if resp.StatusCode != http.StatusOK {
		var ge googlePlacesError
		_ = json.Unmarshal(raw, &ge)
		code := ""
		for _, d := range ge.Error.Details {
			if d.Reason != "" {
				code = d.Reason
				break
			}
		}
		if code == "" {
			code = ge.Error.Status
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return res, &PlacesDeniedError{Reason: deniedReason(code, resp.StatusCode), Code: code}
		}
		return res, fmt.Errorf("places %s: HTTP %d %s", endpoint, resp.StatusCode, code)
	}
	if err = json.Unmarshal(raw, &res); err != nil {
		return res, fmt.Errorf("places %s: unreadable answer: %w", endpoint, err)
	}
	return res, nil
}

// subResult is one search's places and how far its list is complete.
type subResult struct {
	places         []dto.NearbyPlaceItem
	capped         bool
	completeWithin int
}

func (c *PlacesClient) run(ctx context.Context, s placeSearch, cat string, lat float64, lng float64, radiusM int) (subResult, error) {
	got, err := c.call(ctx, s.endpoint, s.body)
	if err != nil {
		return subResult{}, err
	}
	out := subResult{completeWithin: radiusM}
	far := 0.0
	for _, g := range got.Places {
		if g.ID == "" || g.Location == nil {
			continue
		}
		d := geo.HaversineM(lat, lng, g.Location.Latitude, g.Location.Longitude)
		far = math.Max(far, d)
		item := dto.NearbyPlaceItem{
			ID:        "g:" + g.ID,
			PlaceID:   g.ID,
			Name:      strings.TrimSpace(g.DisplayName.Text),
			Category:  cat,
			Lat:       g.Location.Latitude,
			Lng:       g.Location.Longitude,
			DistanceM: int(math.Round(d)),
			Source:    constants.NearbySourceGoogle,
		}
		if s.endpoint == "searchText" && item.DistanceM > radiusM {
			continue // bias circle -> circle
		}
		if kept := s.keep(item, g.PrimaryType); kept != nil {
			out.places = append(out.places, *kept)
		}
	}
	if s.endpoint == "searchText" {
		// one page; nearest first, so a farthest result past the radius
		// means the list is complete
		if got.NextPageToken != "" && int(math.Round(far)) <= radiusM {
			out.capped, out.completeWithin = true, int(math.Floor(far))
		}
		return out, nil
	}
	if len(got.Places) >= placesMaxResults {
		out.capped, out.completeWithin = true, int(math.Floor(far))
	}
	return out, nil
}

// fatalPlacesErr: errors that stop the whole area (the service then uses
// OpenStreetMap for everything).
func fatalPlacesErr(err error) bool {
	var denied *PlacesDeniedError
	return errors.As(err, &denied) || errors.Is(err, ErrPlacesDailyCap)
}

// FetchArea: the office call first and alone (a refused key wastes 1
// call, not 9), then the other categories in parallel, the calls of one
// category one after the other. A failed first category, a refused key or
// the daily cap fails the whole area; other failures go to Failed.
func (c *PlacesClient) FetchArea(ctx context.Context, lat float64, lng float64, radiusM int, brands BrandSet, takeCall func(ctx context.Context) error) (area PlacesArea, err error) {
	ctx, span := placesTracer.Start(ctx, "PlacesClient.FetchArea")
	defer span.End()

	if !c.Configured() {
		return area, ErrPlacesNotSetUp
	}

	var mu sync.Mutex
	calls := 0
	runCategory := func(ctx context.Context, cat string) ([]subResult, error) {
		var subs []subResult
		for _, s := range c.searches(cat, lat, lng, radiusM, brands) {
			if takeCall != nil {
				if err := takeCall(ctx); err != nil {
					if errors.Is(err, ErrPlacesDailyCap) {
						return nil, ErrPlacesDailyCap
					}
					return nil, fmt.Errorf("places: daily cap check: %w", err)
				}
			}
			mu.Lock()
			calls++
			mu.Unlock()
			sub, err := c.run(ctx, s, cat, lat, lng, radiusM)
			if err != nil {
				return nil, err
			}
			subs = append(subs, sub)
		}
		return subs, nil
	}

	cats := c.categories
	if len(cats) == 0 {
		return PlacesArea{Categories: map[string]dto.NearbyCategoryInfo{}}, nil
	}
	first := cats[0]
	for _, cat := range cats {
		if cat == constants.NearbyCategoryOffice {
			first = cat
		}
	}

	results := map[string][]subResult{}
	subs, err := runCategory(ctx, first)
	if err != nil {
		span.RecordError(err)
		return PlacesArea{Calls: calls}, err
	}
	results[first] = subs

	pctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	var fatal error
	failed := map[string]bool{}
	for _, cat := range cats {
		if cat == first {
			continue
		}
		wg.Add(1)
		go func(cat string) {
			defer wg.Done()
			subs, err := runCategory(pctx, cat)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				results[cat] = subs
			case fatalPlacesErr(err):
				if fatal == nil {
					fatal = err
				}
				cancel()
			default:
				failed[cat] = true
			}
		}(cat)
	}
	wg.Wait()
	if fatal != nil {
		span.RecordError(fatal)
		return PlacesArea{Calls: calls}, fatal
	}

	// fixed order, one place once (a brand store found by the brand search
	// and by convenience_store counts once)
	area = PlacesArea{Categories: map[string]dto.NearbyCategoryInfo{}, Calls: calls}
	seen := map[string]bool{}
	for _, cat := range cats {
		if failed[cat] {
			area.Failed = append(area.Failed, cat)
			continue
		}
		info := dto.NearbyCategoryInfo{Source: constants.NearbySourceGoogle}
		cappedAt := -1
		for _, s := range results[cat] {
			if s.capped && (cappedAt < 0 || s.completeWithin < cappedAt) {
				cappedAt = s.completeWithin
			}
			for _, p := range s.places {
				if seen[p.ID] {
					continue
				}
				seen[p.ID] = true
				area.Places = append(area.Places, p)
				info.Count++
			}
		}
		if cappedAt >= 0 {
			v := cappedAt
			info.Capped, info.CompleteWithinM = true, &v
		}
		area.Categories[cat] = info
	}
	span.SetAttributes(attribute.Int("places.calls", calls), attribute.Int("places.count", len(area.Places)))
	return area, nil
}
