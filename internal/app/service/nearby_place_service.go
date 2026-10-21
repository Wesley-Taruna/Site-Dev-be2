package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/client"
	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/repository"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs/env"
	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

// INearbyPlaceService gives the named competitors, offices, schools,
// campuses and apartments around a point (local §6) for the "Pilih titik"
// map and the Gemini prompt. Source: Google Places when a key is set,
// otherwise (or when Google refuses or the daily cap is near) OpenStreetMap;
// the answer always names source and radius. Cost guards: one fetch per
// ~110 m cell, cached 7 days (OSM fallbacks 6 h); a server-wide daily cap on
// Google calls (150; a new area needs 10); after a refused key Google is
// skipped for 1 hour. Off by default ([places] enabled = false: 503, and the
// AI prompt stays as before).
type INearbyPlaceService interface {
	// GetArea is GET /v1/site/nearby-places.
	GetArea(ctx context.Context, req dto.NearbyPlacesRequestGet) (res dto.NearbyAreaResponseGet, err error)
	// Measured is the same data for the prompt at the AI radius; (nil, nil)
	// when switched off.
	Measured(ctx context.Context, lat float64, lng float64) (res *dto.NearbyMeasured, err error)
	// Enabled is [places] enabled.
	Enabled() bool
}

// nearbyConfig is [places] in the config.
type nearbyConfig struct {
	enabled          bool
	aiMeasured       bool
	dailyCap         int64
	defaultRadiusM   int
	maxRadiusM       int
	cacheTTL         time.Duration
	fallbackCacheTTL time.Duration
	aiTimeout        time.Duration
}

func loadNearbyConfig(env env.Provider) nearbyConfig {
	c := nearbyConfig{
		aiMeasured:       true,
		dailyCap:         150,
		defaultRadiusM:   500,
		maxRadiusM:       1000,
		cacheTTL:         7 * 24 * time.Hour,
		fallbackCacheTTL: 6 * time.Hour,
		aiTimeout:        30 * time.Second,
	}
	if env == nil {
		return c
	}
	c.enabled = env.GetBool("places.enabled")
	if env.IsSet("places.ai_measured") {
		c.aiMeasured = env.GetBool("places.ai_measured")
	}
	if n := env.GetInt64("places.daily_cap"); n > 0 {
		c.dailyCap = n
	}
	if n := env.GetInt("places.radius_m"); n > 0 {
		c.defaultRadiusM = n
	}
	if n := env.GetInt("places.max_radius_m"); n > 0 {
		c.maxRadiusM = n
	}
	if n := env.GetInt("places.cache_hours"); n > 0 {
		c.cacheTTL = time.Duration(n) * time.Hour
	}
	if n := env.GetInt("places.fallback_cache_hours"); n > 0 {
		c.fallbackCacheTTL = time.Duration(n) * time.Hour
	}
	if n := env.GetInt("places.ai_timeout_seconds"); n > 0 {
		c.aiTimeout = time.Duration(n) * time.Second
	}
	return c
}

const (
	// nearbyPadM: the cache key rounds the point to ~110 m, so fetch a bit
	// wider and filter from the exact point
	nearbyPadM = 120
	// nearbyCacheVersion: change when the classification changes
	nearbyCacheVersion = "v1"
	// googleOffFor: after Google refuses the key, skip it this long
	googleOffFor = time.Hour
	// storePointsTTL: how long the store list is kept in memory
	storePointsTTL = time.Hour
)

// nearbyState is what all requests of one process share: Google switched
// off after a refusal, one fetch per area at a time, the store list.
type nearbyState struct {
	mu           sync.Mutex
	googleOffAt  time.Time
	googleReason string
	inflight     map[string]*inflightArea
	stores       []storePoint
	storesAt     time.Time
}

type inflightArea struct {
	done chan struct{}
	val  *storedArea
	err  error
}

// sharedNearbyState is the process-wide state (services are built per
// handler; the state must not be).
var sharedNearbyState = &nearbyState{inflight: map[string]*inflightArea{}}

type NearbyPlaceService struct {
	opt                  opt.Options
	RepositoryNearbyArea repository.INearbyAreaRepository
	RepositorySite       repository.ISiteRepository
	ClientPlaces         client.IPlacesClient
	ClientOverpass       client.IOverpassClient
	cfg                  nearbyConfig
	state                *nearbyState
	now                  func() time.Time
}

func NewNearbyPlaceService() INearbyPlaceService {
	return &NearbyPlaceService{
		opt:                  global.Setup.Common,
		RepositoryNearbyArea: repository.NewNearbyAreaRepository(),
		RepositorySite:       repository.NewSiteRepository(),
		ClientPlaces:         global.Setup.Clients.Places,
		ClientOverpass:       global.Setup.Clients.Overpass,
		cfg:                  loadNearbyConfig(global.Setup.Common.Env),
		state:                sharedNearbyState,
		now:                  time.Now,
	}
}

func (s *NearbyPlaceService) Enabled() bool { return s.cfg.enabled }

func (s *NearbyPlaceService) GetArea(ctx context.Context, req dto.NearbyPlacesRequestGet) (res dto.NearbyAreaResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "NearbyPlaceService.GetArea")
	defer span.End()

	res, err = s.area(ctx, req.Lat, req.Lng, req.RadiusM)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		res = dto.NearbyAreaResponseGet{}
		return
	}

	return
}

func (s *NearbyPlaceService) area(ctx context.Context, lat float64, lng float64, radiusM int) (res dto.NearbyAreaResponseGet, err error) {
	if !s.cfg.enabled {
		err = echo.NewHTTPError(http.StatusServiceUnavailable, "Data sekitar titik belum diaktifkan di server ini ([places] enabled = false).")
		return
	}
	if err = validatePointInIndonesia(lat, lng); err != nil {
		return
	}
	if radiusM <= 0 {
		radiusM = s.cfg.defaultRadiusM
	}
	if radiusM > s.cfg.maxRadiusM {
		err = echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("radius maksimal %d m", s.cfg.maxRadiusM))
		return
	}

	cLat := math.Round(lat*1000) / 1000
	cLng := math.Round(lng*1000) / 1000
	key := fmt.Sprintf("%s:%.3f,%.3f:%d", nearbyCacheVersion, cLat, cLng, radiusM)

	st, cached, err := s.loadArea(ctx, key, cLat, cLng, radiusM+nearbyPadM)
	if err != nil {
		return
	}

	places, categories := areaForPoint(st, lat, lng, radiusM)
	res = dto.NearbyAreaResponseGet{
		Source:         st.Source,
		Lat:            lat,
		Lng:            lng,
		RadiusM:        radiusM,
		FetchedAt:      time.Unix(st.FetchedAtUnix, 0),
		Places:         places,
		Categories:     categories,
		Cached:         cached,
		FallbackReason: st.FallbackReason,
		StoreSource:    "real",
		Attribution:    nearbyAttribution(st.Source),
	}
	if !cached {
		res.GoogleCalls = st.GoogleCalls
	}
	if stores, storeErr := s.storePoints(ctx); storeErr == nil {
		res.NearestFamilyMart = nearestStore(lat, lng, stores)
	} else {
		s.opt.Logger.AddMessage(log.ErrorLevel, fmt.Errorf("nearby_place_service: store list unavailable: %w", storeErr)).Print()
	}
	return
}

func (s *NearbyPlaceService) Measured(ctx context.Context, lat float64, lng float64) (res *dto.NearbyMeasured, err error) {
	if !s.cfg.enabled || !s.cfg.aiMeasured {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, s.cfg.aiTimeout)
	defer cancel()

	area, err := s.GetArea(ctx, dto.NearbyPlacesRequestGet{Lat: lat, Lng: lng, RadiusM: s.cfg.defaultRadiusM})
	if err != nil {
		return nil, err
	}
	return measuredFromArea(area), nil
}

// loadArea: cache, else one fetch per key at a time.
func (s *NearbyPlaceService) loadArea(ctx context.Context, key string, lat float64, lng float64, fetchR int) (*storedArea, bool, error) {
	if raw, ok := s.RepositoryNearbyArea.GetArea(ctx, key); ok {
		var st storedArea
		if json.Unmarshal(raw, &st) == nil {
			return &st, true, nil
		}
	}

	s.state.mu.Lock()
	if f, ok := s.state.inflight[key]; ok {
		s.state.mu.Unlock()
		select {
		case <-f.done:
			return f.val, true, f.err
		case <-ctx.Done():
			return nil, false, ctx.Err()
		}
	}
	f := &inflightArea{done: make(chan struct{})}
	s.state.inflight[key] = f
	s.state.mu.Unlock()

	f.val, f.err = s.build(ctx, lat, lng, fetchR)
	s.state.mu.Lock()
	delete(s.state.inflight, key)
	s.state.mu.Unlock()
	close(f.done)

	if f.err != nil {
		return nil, false, f.err
	}
	ttl := s.cfg.cacheTTL
	if f.val.FallbackReason != "" {
		ttl = s.cfg.fallbackCacheTTL
	}
	if raw, err := json.Marshal(f.val); err == nil {
		s.RepositoryNearbyArea.SetArea(ctx, key, raw, ttl)
	}
	return f.val, false, nil
}

// googleUsable: a key, not refused within the last hour, enough calls left.
func (s *NearbyPlaceService) googleUsable(ctx context.Context) (bool, string) {
	if !s.ClientPlaces.Configured() {
		return false, "Kunci Google Places belum diisi di server ini"
	}
	s.state.mu.Lock()
	off, reason := s.state.googleOffAt, s.state.googleReason
	s.state.mu.Unlock()
	if !off.IsZero() && s.now().Before(off.Add(googleOffFor)) {
		return false, reason
	}
	if s.RepositoryNearbyArea.GoogleCallsLeft(ctx, s.cfg.dailyCap) < client.GooglePlacesCallsPerArea {
		return false, "Batas harian panggilan Google hampir habis"
	}
	return true, ""
}

// build fetches one area: Google first when usable, else OpenStreetMap.
func (s *NearbyPlaceService) build(ctx context.Context, lat float64, lng float64, fetchR int) (*storedArea, error) {
	st := &storedArea{FetchedAtUnix: s.now().Unix(), Lat: lat, Lng: lng, FetchRadiusM: fetchR}

	usable, reason := s.googleUsable(ctx)
	if usable {
		ga, err := s.ClientPlaces.FetchArea(ctx, lat, lng, fetchR, func(ctx context.Context) error {
			if err := s.RepositoryNearbyArea.TakeGoogleCall(ctx, s.cfg.dailyCap); err != nil {
				if errors.Is(err, repository.ErrNearbyDailyCap) {
					return client.ErrPlacesDailyCap
				}
				return err
			}
			return nil
		})
		st.GoogleCalls = ga.Calls
		if err == nil {
			st.Source = constants.NearbySourceGoogle
			st.Places = ga.Places
			st.Categories = ga.Categories
			if st.Categories == nil {
				st.Categories = map[string]dto.NearbyCategoryInfo{}
			}
			if len(ga.Failed) > 0 {
				s.fillFromOSM(ctx, st, ga.Failed)
			}
			return st, nil
		}
		var denied *client.PlacesDeniedError
		switch {
		case errors.As(err, &denied):
			s.state.mu.Lock()
			s.state.googleOffAt, s.state.googleReason = s.now(), denied.Reason
			s.state.mu.Unlock()
			reason = denied.Reason
		case errors.Is(err, client.ErrPlacesDailyCap):
			reason = "Batas harian panggilan Google tercapai"
		default:
			reason = "Google Places gagal dihubungi"
		}
		s.opt.Logger.AddMessage(log.ErrorLevel, fmt.Errorf("nearby_place_service: google not used (%s): %w", reason, err)).Print()
	}

	places, err := s.ClientOverpass.FetchArea(ctx, lat, lng, fetchR)
	if err != nil {
		s.opt.Logger.AddMessage(log.ErrorLevel, err).Print()
		return nil, echo.NewHTTPError(http.StatusBadGateway, "Data peta (OpenStreetMap) tidak bisa diambil sekarang. Coba lagi sebentar lagi.")
	}
	st.Source = constants.NearbySourceOSM
	st.Places = places
	st.Categories = map[string]dto.NearbyCategoryInfo{}
	for _, c := range constants.NearbyCategories {
		st.Categories[c] = dto.NearbyCategoryInfo{Source: constants.NearbySourceOSM}
	}
	st.FallbackReason = reason
	return st, nil
}

// fillFromOSM replaces the Google categories that failed with OSM data
// (source "mixed", labelled per category).
func (s *NearbyPlaceService) fillFromOSM(ctx context.Context, st *storedArea, failed []string) {
	osm, err := s.ClientOverpass.FetchArea(ctx, st.Lat, st.Lng, st.FetchRadiusM)
	if err != nil {
		for _, c := range failed {
			zero := 0
			st.Categories[c] = dto.NearbyCategoryInfo{Source: constants.NearbySourceGoogle, Capped: true, CompleteWithinM: &zero}
		}
		return
	}
	for _, c := range failed {
		info := dto.NearbyCategoryInfo{Source: constants.NearbySourceOSM}
		for _, p := range osm {
			if p.Category == c {
				st.Places = append(st.Places, p)
				info.Count++
			}
		}
		st.Categories[c] = info
	}
	st.Source = constants.NearbySourceMixed
}

// storePoints is the active store list, kept in memory for an hour.
func (s *NearbyPlaceService) storePoints(ctx context.Context) ([]storePoint, error) {
	s.state.mu.Lock()
	if s.state.stores != nil && s.now().Sub(s.state.storesAt) < storePointsTTL {
		v := s.state.stores
		s.state.mu.Unlock()
		return v, nil
	}
	s.state.mu.Unlock()

	rows, err := s.RepositorySite.GetActiveStorePoints(ctx)
	if err != nil {
		return nil, err
	}
	points := make([]storePoint, 0, len(rows))
	for _, r := range rows {
		points = append(points, storePoint{Code: r.StoreCode, Name: r.StoreName, Lat: r.Latitude, Lng: r.Longitude})
	}

	s.state.mu.Lock()
	s.state.stores, s.state.storesAt = points, s.now()
	s.state.mu.Unlock()
	return points, nil
}

// insightDataSource labels which map data went into the prompt.
func insightDataSource(m *dto.NearbyMeasured) string {
	if m == nil {
		return "gemini"
	}
	return "gemini+" + m.Source
}
