package client

import (
	"context"
	"errors"

	dto "site-dev-backend/internal/app/dto"

	"gitlab.com/fami-app-v2/familib/familabs/env"
)

// ErrOverpassNotSetUp is returned until the real OpenStreetMap client exists.
var ErrOverpassNotSetUp = errors.New("overpass: OpenStreetMap client is not set up yet (TODO, plan Q4)")

// IOverpassClient is OpenStreetMap through the Overpass API (free, no key):
// the fallback source of nearby places.
type IOverpassClient interface {
	// FetchArea returns every counted OSM place within radiusM, nearest
	// first (OverpassAreaQuery + ParseOverpassArea).
	FetchArea(ctx context.Context, lat float64, lng float64, radiusM int) (places []dto.NearbyPlaceItem, err error)
}

// NewOverpassClient returns the OpenStreetMap client.
//
// TODO(plan Q4, needs Wes's OK): the HTTP calls. Local §6 (site-dev-api
// 2191e07 places_overpass.go, standard library only) posts
// OverpassAreaQuery as form field "data" to each server in turn until one
// answers 200 (places.overpass_urls, default maps.mail.ru/osm/tools/overpass/
// api/interpreter, overpass-api.de/api/interpreter,
// overpass.kumi.systems/api/interpreter), with User-Agent
// places.overpass_user_agent (default "site-dev-api (FamilyMart Indonesia
// site survey; contact: Site Development)"), Accept application/json, a
// per-server timeout (places.overpass_timeout_seconds, 25), at most one
// request per 1.1 s per process, answers up to 20 MB; then
// ParseOverpassArea. Until built it answers ErrOverpassNotSetUp.
func NewOverpassClient(_ env.Provider) IOverpassClient {
	return pendingOverpassClient{}
}

type pendingOverpassClient struct{}

func (pendingOverpassClient) FetchArea(context.Context, float64, float64, int) ([]dto.NearbyPlaceItem, error) {
	return nil, ErrOverpassNotSetUp
}
