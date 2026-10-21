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
	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type IGeocodingService interface {
	// Reverse is GET /v1/geocoding/reverse (one from the daily quota).
	Reverse(ctx context.Context, lat float64, lng float64, actorID int64) (res dto.ReverseGeocodeResponse, err error)
	// Search is GET /v1/geocoding/search (one from the daily quota).
	Search(ctx context.Context, query string, actorID int64) (res []dto.AddressSearchResult, err error)
}

type GeocodingService struct {
	opt             opt.Options
	RepositoryQuota repository.IQuotaRepository
	ClientGeocoding client.IGeocodingClient
}

func NewGeocodingService() IGeocodingService {
	return &GeocodingService{
		opt:             global.Setup.Common,
		RepositoryQuota: repository.NewQuotaRepository(),
		ClientGeocoding: global.Setup.Clients.Geocoding,
	}
}

func (s *GeocodingService) Reverse(ctx context.Context, lat float64, lng float64, actorID int64) (res dto.ReverseGeocodeResponse, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "GeocodingService.Reverse")
	defer span.End()

	if err = s.useQuota(ctx, actorID); err == nil {
		res, err = s.ClientGeocoding.Reverse(ctx, lat, lng)
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
	}
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		res = nil
		return
	}

	return
}

// useQuota counts one Google call; used up = 429 "Kuota reverse-geocode
// harian (<limit>) sudah habis, coba lagi besok."
func (s *GeocodingService) useQuota(ctx context.Context, actorID int64) error {
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
