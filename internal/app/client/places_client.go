package client

import (
	"context"
	"errors"

	dto "site-dev-backend/internal/app/dto"

	"gitlab.com/fami-app-v2/familib/familabs/env"
)

// GooglePlacesCallsPerArea is the number of Google calls for one new area.
const GooglePlacesCallsPerArea = 10

var (
	// ErrPlacesNotSetUp is returned until the real Google Places client exists.
	ErrPlacesNotSetUp = errors.New("places: Google Places client is not set up yet (TODO, plan Q4)")
	// ErrPlacesDailyCap: the server-wide daily cap of Google calls is reached.
	ErrPlacesDailyCap = errors.New("places: daily Google call cap reached")
)

// PlacesDeniedError: Google refused the key for Places (API disabled, key
// restriction); Reason is plain Indonesian for the answer.
type PlacesDeniedError struct{ Reason string }

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
	// FetchArea fetches every category within radiusM; takeCall is called
	// before each Google call (daily cap) and its error stops the fetch.
	FetchArea(ctx context.Context, lat float64, lng float64, radiusM int, takeCall func(ctx context.Context) error) (area PlacesArea, err error)
}

// NewPlacesClient returns the Google Places client.
//
// TODO(plan Q4, needs Wes's OK): the HTTP calls. Local §6 (site-dev-api
// 2191e07 places_google.go, standard library only) makes 10 calls per new
// area to https://places.googleapis.com/v1/places:searchText and
// :searchNearby with headers X-Goog-Api-Key (never in the URL) and
// X-Goog-FieldMask "places.id,places.displayName,places.location,
// places.types,places.primaryType" (+ ",nextPageToken" for text search),
// languageCode "id", regionCode "ID": office first and alone (corporate_office;
// a refused key wastes 1 call, not 10), then competitors (text search
// Indomaret, Alfamart, Alfamidi, Lawson, Circle K in a rectangle, pageSize
// 20, rank DISTANCE, kept only when BrandOf(name) is that chain and
// !IsNotAShop; plus nearby convenience_store kept for any chain), schools
// (school, primary_school, secondary_school, preschool), campuses
// (university), apartments (apartment_building, apartment_complex,
// condominium_complex); nearby search max 20 in a circle, rank DISTANCE,
// FamilyMart names dropped. 20 results (or a nextPageToken) = capped,
// complete up to the farthest result. 401/403 = PlacesDeniedError (reason
// from error.details[].reason: API_KEY_SERVICE_BLOCKED, SERVICE_DISABLED);
// a failed office call or a denial fails the whole area, other failures go
// to Failed. Place ids "g:<id>", PlaceID set, Source "google". Until built,
// FetchArea answers ErrPlacesNotSetUp and the service uses OpenStreetMap.
func NewPlacesClient(env env.Provider) IPlacesClient {
	c := pendingPlacesClient{}
	if env != nil {
		c.configured = env.GetString("places.google_apikey") != ""
	}
	return c
}

type pendingPlacesClient struct{ configured bool }

func (c pendingPlacesClient) Configured() bool { return c.configured }

func (pendingPlacesClient) FetchArea(context.Context, float64, float64, int, func(context.Context) error) (PlacesArea, error) {
	return PlacesArea{}, ErrPlacesNotSetUp
}
