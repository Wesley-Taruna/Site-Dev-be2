package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Setiabudi, Jakarta
const fmLat, fmLng = -6.2090, 106.8298

func newTestFMStores(t *testing.T) (*FamilyMartStoreService, *mocks.IFamilyMartStoreRepository, *mocks.INearbyAreaRepository) {
	store := mocks.NewIFamilyMartStoreRepository(t)
	cache := mocks.NewINearbyAreaRepository(t)
	store.On("CacheTag").Return("tag1").Maybe()
	return &FamilyMartStoreService{opt: testOptions(), RepositoryStore: store, RepositoryCache: cache, cfg: loadFamilyMartStoreConfig(nil)}, store, cache
}

func addr(s string) *string { return &s }

// fmRows: stores at a known offset north of the point (1 deg lat ~ 111.2 km).
func fmRows() []model.FamilyMartStore {
	return []model.FamilyMartStore{
		{StoreCode: "F900", StoreName: "Jauh 1,5 km", Latitude: fmLat + 0.0135, Longitude: fmLng},
		{StoreCode: "F300", StoreName: " Tiga Ratus ", Address: addr(" Jl. Tiga No. 3 "), Latitude: fmLat + 0.0027, Longitude: fmLng},
		{StoreCode: "F100", StoreName: "Seratus", Address: addr("Jl. Satu"), Latitude: fmLat + 0.0009, Longitude: fmLng},
		{StoreCode: "F950", StoreName: "Sudut kotak 1,3 km", Latitude: fmLat + 0.0089, Longitude: fmLng + 0.0089},
		{StoreCode: "F000", StoreName: "Tanpa koordinat", Latitude: 0, Longitude: 0},
		{StoreCode: "F100", StoreName: "Seratus (dobel)", Latitude: fmLat + 0.0009, Longitude: fmLng},
		{StoreCode: "F800", StoreName: "Delapan Ratus", Latitude: fmLat - 0.0072, Longitude: fmLng},
	}
}

func TestFMStoresNear_DistanceFilterOrder(t *testing.T) {
	s, store, cache := newTestFMStores(t)
	key := "fmstores:v1:tag1:-6.209,106.830:1120"
	cache.On("GetArea", ctxAny, key).Return(nil, false).Once()
	store.On("InBox", ctxAny, mock.Anything, mock.Anything, mock.Anything, mock.Anything, fmStoresSQLCap).
		Return(func(_ context.Context, minLat, maxLat, minLng, maxLng float64, _ int) ([]model.FamilyMartStore, error) {
			// box of 1120 m around the cell centre
			assert.InDelta(t, -6.209-0.01006, minLat, 0.0001)
			assert.InDelta(t, -6.209+0.01006, maxLat, 0.0001)
			assert.Less(t, minLng, 106.830-0.01006)
			assert.Greater(t, maxLng, 106.830+0.01006)
			return fmRows(), nil
		}).Once()
	cache.On("SetArea", ctxAny, key, mock.Anything, time.Hour).Return().Once()

	res, err := s.Near(context.Background(), dto.FamilyMartStoresRequestGet{Lat: fmLat, Lng: fmLng})

	require.NoError(t, err)
	assert.Equal(t, 1000, res.RadiusM)
	assert.False(t, res.Cached)
	assert.False(t, res.Capped)
	require.Equal(t, 3, res.Count)
	require.Len(t, res.Stores, 3)
	assert.Equal(t, dto.FamilyMartStoreItem{StoreCode: "F100", StoreName: "Seratus", Address: "Jl. Satu", Lat: fmLat + 0.0009, Lng: fmLng, DistanceM: 100}, res.Stores[0])
	assert.Equal(t, "F300", res.Stores[1].StoreCode)
	assert.Equal(t, "Tiga Ratus", res.Stores[1].StoreName)
	assert.Equal(t, "Jl. Tiga No. 3", res.Stores[1].Address)
	assert.Equal(t, 300, res.Stores[1].DistanceM)
	assert.Equal(t, "F800", res.Stores[2].StoreCode)
	assert.Equal(t, "", res.Stores[2].Address)
	assert.Equal(t, 801, res.Stores[2].DistanceM)
}

func TestFMStoresNear_CachedRowsFilteredFromExactPoint(t *testing.T) {
	s, _, cache := newTestFMStores(t)
	raw, err := json.Marshal(fmRows())
	require.NoError(t, err)
	cache.On("GetArea", ctxAny, "fmstores:v1:tag1:-6.209,106.830:620").Return(raw, true).Once()

	res, err := s.Near(context.Background(), dto.FamilyMartStoresRequestGet{Lat: fmLat, Lng: fmLng, RadiusM: 500})

	require.NoError(t, err)
	assert.True(t, res.Cached)
	assert.Equal(t, 500, res.RadiusM)
	require.Len(t, res.Stores, 2)
	assert.Equal(t, []string{"F100", "F300"}, []string{res.Stores[0].StoreCode, res.Stores[1].StoreCode})
}

func TestFMStoresNear_EmptyIsListNotNull(t *testing.T) {
	s, store, cache := newTestFMStores(t)
	cache.On("GetArea", ctxAny, mock.Anything).Return(nil, false).Once()
	store.On("InBox", ctxAny, mock.Anything, mock.Anything, mock.Anything, mock.Anything, fmStoresSQLCap).Return(nil, nil).Once()
	cache.On("SetArea", ctxAny, mock.Anything, []byte("[]"), time.Hour).Return().Once()

	res, err := s.Near(context.Background(), dto.FamilyMartStoresRequestGet{Lat: fmLat, Lng: fmLng})

	require.NoError(t, err)
	b, err := json.Marshal(res)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"stores":[]`)
	assert.Equal(t, 0, res.Count)
}

func TestFMStoresNear_CapKeepsNearest(t *testing.T) {
	s, store, cache := newTestFMStores(t)
	s.cfg.maxResults = 2
	s.cfg.cacheTTL = 0 // no cache: GetArea/SetArea never called
	store.On("InBox", ctxAny, mock.Anything, mock.Anything, mock.Anything, mock.Anything, fmStoresSQLCap).Return(fmRows(), nil).Once()

	res, err := s.Near(context.Background(), dto.FamilyMartStoresRequestGet{Lat: fmLat, Lng: fmLng})

	require.NoError(t, err)
	assert.True(t, res.Capped)
	assert.Equal(t, 2, res.Count)
	assert.Equal(t, "F100", res.Stores[0].StoreCode)
	assert.Equal(t, "F300", res.Stores[1].StoreCode)
	cache.AssertNotCalled(t, "GetArea", mock.Anything, mock.Anything)
}

func TestFMStoresNear_Validation(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  dto.FamilyMartStoresRequestGet
		msg  string
	}{
		{"outside Indonesia", dto.FamilyMartStoresRequestGet{Lat: 51.5, Lng: -0.12}, "Koordinat tidak valid (harus di Indonesia)."},
		{"0,0", dto.FamilyMartStoresRequestGet{}, "Koordinat tidak valid (harus di Indonesia)."},
		{"radius too big", dto.FamilyMartStoresRequestGet{Lat: fmLat, Lng: fmLng, RadiusM: 1001}, "radius maksimal 1000 m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := newTestFMStores(t)
			_, err := s.Near(context.Background(), tc.req)
			var he *echo.HTTPError
			require.True(t, errors.As(err, &he))
			assert.Equal(t, http.StatusBadRequest, he.Code)
			assert.Equal(t, tc.msg, he.Message)
		})
	}
}

func TestFMStoresNear_DatabaseErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		msg  string
	}{
		{"bad config", fmt.Errorf("%w: table %q", repository.ErrStoreDBConfig, "x;y"), "Data toko FamilyMart belum diatur dengan benar di server ini ([store_database])."},
		{"unreachable", repository.ErrStoreDBUnavailable, "Data toko FamilyMart tidak bisa dibaca sekarang. Coba lagi sebentar lagi."},
		{"query error", errors.New("mssql: Invalid object name"), "Data toko FamilyMart tidak bisa dibaca sekarang. Coba lagi sebentar lagi."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, store, cache := newTestFMStores(t)
			cache.On("GetArea", ctxAny, mock.Anything).Return(nil, false).Once()
			store.On("InBox", ctxAny, mock.Anything, mock.Anything, mock.Anything, mock.Anything, fmStoresSQLCap).Return(nil, tc.err).Once()

			_, err := s.Near(context.Background(), dto.FamilyMartStoresRequestGet{Lat: fmLat, Lng: fmLng})

			var he *echo.HTTPError
			require.True(t, errors.As(err, &he))
			assert.Equal(t, http.StatusServiceUnavailable, he.Code)
			assert.Equal(t, tc.msg, he.Message)
		})
	}
}

func TestFMStoresConfig_DefaultsAndLimits(t *testing.T) {
	c := loadFamilyMartStoreConfig(nil)
	assert.Equal(t, familyMartStoreConfig{defaultRadiusM: 1000, maxRadiusM: 1000, maxResults: 200, cacheTTL: time.Hour}, c)
}
