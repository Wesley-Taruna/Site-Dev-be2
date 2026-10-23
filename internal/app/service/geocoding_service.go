package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

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

type IGeocodingService interface {
	// Reverse is GET /v1/geocoding/reverse (one from the daily quota).
	Reverse(ctx context.Context, lat float64, lng float64, actorID int64) (res dto.ReverseGeocodeResponse, err error)
	// Search is GET /v1/geocoding/search (one from the daily quota).
	Search(ctx context.Context, query string, actorID int64) (res []dto.AddressSearchResult, err error)
}

// geocodingConfig is [geocoding]: enabled (default false: 503, no quota
// used, no Google call) and daily_cap (server-wide Google calls per day,
// default 150, on top of the per-user daily_quota).
type geocodingConfig struct {
	enabled  bool
	dailyCap int64
}

const defaultGeocodingDailyCap = 150

func loadGeocodingConfig(env env.Provider) geocodingConfig {
	c := geocodingConfig{dailyCap: defaultGeocodingDailyCap}
	if env == nil {
		return c
	}
	c.enabled = env.GetBool("geocoding.enabled")
	if n := env.GetInt64("geocoding.daily_cap"); n > 0 {
		c.dailyCap = n
	}
	return c
}

type GeocodingService struct {
	opt                    opt.Options
	RepositoryQuota        repository.IQuotaRepository
	RepositoryGoogleBudget repository.IGoogleBudgetRepository
	ClientGeocoding        client.IGeocodingClient
	cfg                    geocodingConfig
}

func NewGeocodingService() IGeocodingService {
	return &GeocodingService{
		opt:                    global.Setup.Common,
		RepositoryQuota:        repository.NewQuotaRepository(),
		RepositoryGoogleBudget: repository.NewGoogleBudgetRepository(),
		ClientGeocoding:        global.Setup.Clients.Geocoding,
		cfg:                    loadGeocodingConfig(global.Setup.Common.Env),
	}
}

func (s *GeocodingService) Reverse(ctx context.Context, lat float64, lng float64, actorID int64) (res dto.ReverseGeocodeResponse, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "GeocodingService.Reverse")
	defer span.End()

	if err = s.useQuota(ctx, actorID); err == nil {
		res, err = s.ClientGeocoding.Reverse(ctx, lat, lng)
		err = geocodingDenied503(err)
	}
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		res = dto.ReverseGeocodeResponse{}
		return
	}

	return
}

func (s *GeocodingService) Search(ctx context.Context, query string, actorID int64) (res []dto.AddressSearchResult, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "GeocodingService.Search")
	defer span.End()

	if err = s.useQuota(ctx, actorID); err == nil {
		res, err = s.ClientGeocoding.Search(ctx, query)
		err = geocodingDenied503(err)
	}
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		res = nil
		return
	}

	return
}

// useQuota checks the switch, then counts one Google call server-wide
// (daily_cap) and for the user; switched off or no key = 503, cap reached = 429
// "Batas harian ...", user quota used up = 429 "Kuota reverse-geocode
// harian (<limit>) sudah habis, coba lagi besok."
func (s *GeocodingService) useQuota(ctx context.Context, actorID int64) error {
	if !s.cfg.enabled {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Alamat otomatis (Google Geocoding) belum diaktifkan di server ini ([geocoding] enabled = false).")
	}
	if !s.ClientGeocoding.Configured() {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Kunci Google Geocoding belum diisi di server ini (geocoding.apikey).")
	}
	if err := s.RepositoryGoogleBudget.Take(ctx, constants.QuotaGeocoding, s.cfg.dailyCap); err != nil {
		if errors.Is(err, repository.ErrGoogleDailyCap) {
			return echo.NewHTTPError(http.StatusTooManyRequests, "Batas harian panggilan Google Geocoding di server ini tercapai, coba lagi besok.")
		}
		return err
	}
	_, err := s.RepositoryQuota.Consume(ctx, constants.QuotaGeocoding, actorID)
	if errors.Is(err, repository.ErrGeocodingQuotaExceeded) {
		limit, limitErr := s.RepositoryQuota.Limit(ctx, constants.QuotaGeocoding)
		if limitErr != nil {
			return limitErr
		}
		return echo.NewHTTPError(http.StatusTooManyRequests, fmt.Sprintf("Kuota reverse-geocode harian (%d) sudah habis, coba lagi besok.", limit))
	}
	return err
}

// geocodingDenied503: Google refused the key (REQUEST_DENIED / 401 / 403)
// = 503 "not available" like no key, not 500 (23 Oct); the answer names
// Google's status word only.
func geocodingDenied503(err error) error {
	var denied *client.GeocodingDeniedError
	if errors.As(err, &denied) {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Alamat otomatis belum tersedia: Google menolak kunci Geocoding di server ini ("+denied.Status+").")
	}
	return err
}
