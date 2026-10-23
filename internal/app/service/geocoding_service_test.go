package service

import (
	"context"
	"net/http"
	"testing"

	"site-dev-backend/internal/app/client"
	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/app/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGeocoding(t *testing.T, enabled bool) (*GeocodingService, *mocks.IQuotaRepository, *mocks.IGoogleBudgetRepository, *mocks.IGeocodingClient) {
	q, b, c := mocks.NewIQuotaRepository(t), mocks.NewIGoogleBudgetRepository(t), mocks.NewIGeocodingClient(t)
	cfg := loadGeocodingConfig(nil)
	cfg.enabled = enabled
	return &GeocodingService{opt: testOptions(), RepositoryQuota: q, RepositoryGoogleBudget: b, ClientGeocoding: c, cfg: cfg}, q, b, c
}

func TestGeocoding_OffIs503WithoutQuotaOrCall(t *testing.T) {
	s, _, _, _ := newTestGeocoding(t, false) // mocks: no call expected

	_, err := s.Reverse(context.Background(), -6.26, 106.81, 7)
	assert.Equal(t, http.StatusServiceUnavailable, httpCode(t, err))
	_, err = s.Search(context.Background(), "Kemang", 7)
	assert.Equal(t, http.StatusServiceUnavailable, httpCode(t, err))
}

func TestGeocoding_NoKeyIs503WithoutQuota(t *testing.T) {
	s, _, _, c := newTestGeocoding(t, true)
	c.On("Configured").Return(false)

	_, err := s.Reverse(context.Background(), -6.26, 106.81, 7)

	assert.Equal(t, http.StatusServiceUnavailable, httpCode(t, err))
	assert.Contains(t, err.Error(), "Kunci Google Geocoding belum diisi")
}

func TestGeocoding_ServerCapThenUserQuotaThenGoogle(t *testing.T) {
	s, q, b, c := newTestGeocoding(t, true)
	assert.Equal(t, int64(150), s.cfg.dailyCap)
	c.On("Configured").Return(true)

	b.On("Take", ctxAny, constants.QuotaGeocoding, int64(150)).Return(repository.ErrGoogleDailyCap).Once()
	_, err := s.Reverse(context.Background(), -6.26, 106.81, 7)
	assert.Equal(t, http.StatusTooManyRequests, httpCode(t, err))
	assert.Contains(t, err.Error(), "Batas harian panggilan Google Geocoding")

	b.On("Take", ctxAny, constants.QuotaGeocoding, int64(150)).Return(nil)
	q.On("Consume", ctxAny, constants.QuotaGeocoding, int64(7)).Return(int64(99), nil).Once()
	c.On("Reverse", ctxAny, -6.26, 106.81).Return(dto.ReverseGeocodeResponse{Address1: "Jalan Kemang Raya No. 1", PostalCode: "12730"}, nil).Once()
	res, err := s.Reverse(context.Background(), -6.26, 106.81, 7)
	require.NoError(t, err)
	assert.Equal(t, "12730", res.PostalCode)

	q.On("Consume", ctxAny, constants.QuotaGeocoding, int64(7)).Return(int64(0), repository.ErrGeocodingQuotaExceeded).Once()
	q.On("Limit", ctxAny, constants.QuotaGeocoding).Return(int64(100), nil).Once()
	_, err = s.Search(context.Background(), "Kemang", 7)
	assert.Equal(t, http.StatusTooManyRequests, httpCode(t, err))
	assert.Contains(t, err.Error(), "Kuota reverse-geocode harian (100)")
}

// 23 Oct: Google refusing the key = 503 "not available" (was 500).
func TestGeocoding_RefusedKeyIs503(t *testing.T) {
	s, q, b, c := newTestGeocoding(t, true)
	c.On("Configured").Return(true)
	b.On("Take", ctxAny, constants.QuotaGeocoding, int64(150)).Return(nil)
	q.On("Consume", ctxAny, constants.QuotaGeocoding, int64(7)).Return(int64(99), nil)
	c.On("Reverse", ctxAny, -6.26, 106.81).Return(dto.ReverseGeocodeResponse{}, &client.GeocodingDeniedError{Status: "REQUEST_DENIED", Message: "API not authorized"}).Once()
	c.On("Search", ctxAny, "Kemang").Return(nil, &client.GeocodingDeniedError{Status: "REQUEST_DENIED"}).Once()

	_, err := s.Reverse(context.Background(), -6.26, 106.81, 7)
	assert.Equal(t, http.StatusServiceUnavailable, httpCode(t, err))
	assert.Contains(t, err.Error(), "Google menolak kunci Geocoding di server ini (REQUEST_DENIED)")
	assert.NotContains(t, err.Error(), "API not authorized", "Google's message stays out of the answer")
	_, err = s.Search(context.Background(), "Kemang", 7)
	assert.Equal(t, http.StatusServiceUnavailable, httpCode(t, err))
}
