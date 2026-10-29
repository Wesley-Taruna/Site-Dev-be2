package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"site-dev-backend/global"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs/env"
	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

// IFamilyMartStoreService gives the active FamilyMart stores within 1 km
// of a point, nearest first, to draw on the map (Wes, 29 Oct). Data: the
// FamilyMart store database ([store_database]; default the main connection
// and AMFIBI_MASTER.dbo.MStore, like /v1/site/maps). No outside service,
// no cost. The rows of a ~110 m cell are kept in Redis for cache_minutes.
type IFamilyMartStoreService interface {
	// Near is GET /v1/site/familymart-stores.
	Near(ctx context.Context, req dto.FamilyMartStoresRequestGet) (res dto.FamilyMartStoresResponseGet, err error)
}

// familyMartStoreConfig is [familymart_stores].
type familyMartStoreConfig struct {
	defaultRadiusM int
	maxRadiusM     int
	maxResults     int
	cacheTTL       time.Duration
}

func loadFamilyMartStoreConfig(e env.Provider) familyMartStoreConfig {
	c := familyMartStoreConfig{defaultRadiusM: 1000, maxRadiusM: 1000, maxResults: 200, cacheTTL: time.Hour}
	if e == nil {
		return c
	}
	if n := e.GetInt("familymart_stores.radius_m"); n > 0 {
		c.defaultRadiusM = n
	}
	if n := e.GetInt("familymart_stores.max_radius_m"); n > 0 {
		c.maxRadiusM = n
	}
	if n := e.GetInt("familymart_stores.max_results"); n > 0 {
		c.maxResults = n
	}
	if e.IsSet("familymart_stores.cache_minutes") {
		c.cacheTTL = time.Duration(e.GetInt("familymart_stores.cache_minutes")) * time.Minute
	}
	if c.defaultRadiusM > c.maxRadiusM {
		c.defaultRadiusM = c.maxRadiusM
	}
	return c
}

const (
	// fmStoresCacheVersion: change when the cached row shape changes
	fmStoresCacheVersion = "fmstores:v1"
	// fmStoresPadM: the cache cell centre is up to ~80 m from the point,
	// so the box is read a bit wider and filtered from the exact point
	fmStoresPadM = 120
	// fmStoresSQLCap: most rows read from one box (a 1.1 km box holds a
	// few dozen stores in central Jakarta; this only guards the database)
	fmStoresSQLCap = 2000
)

type FamilyMartStoreService struct {
	opt             opt.Options
	RepositoryStore repository.IFamilyMartStoreRepository
	RepositoryCache repository.INearbyAreaRepository
	cfg             familyMartStoreConfig
}

func NewFamilyMartStoreService() IFamilyMartStoreService {
	return &FamilyMartStoreService{
		opt:             global.Setup.Common,
		RepositoryStore: repository.NewFamilyMartStoreRepository(),
		RepositoryCache: repository.NewNearbyAreaRepository(),
		cfg:             loadFamilyMartStoreConfig(global.Setup.Common.Env),
	}
}

func (s *FamilyMartStoreService) Near(ctx context.Context, req dto.FamilyMartStoresRequestGet) (res dto.FamilyMartStoresResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "FamilyMartStoreService.Near")
	defer span.End()

	if err = validatePointInIndonesia(req.Lat, req.Lng); err != nil {
		return
	}
	radiusM := req.RadiusM
	if radiusM <= 0 {
		radiusM = s.cfg.defaultRadiusM
	}
	if radiusM > s.cfg.maxRadiusM {
		err = echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("radius maksimal %d m", s.cfg.maxRadiusM))
		return
	}

	rows, cached, err := s.cellRows(ctx, req.Lat, req.Lng, radiusM)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, fmt.Errorf("familymart_store_service: %w", err)).Print()
		if errors.Is(err, repository.ErrStoreDBConfig) {
			err = echo.NewHTTPError(http.StatusServiceUnavailable, "Data toko FamilyMart belum diatur dengan benar di server ini ([store_database]).")
		} else {
			err = echo.NewHTTPError(http.StatusServiceUnavailable, "Data toko FamilyMart tidak bisa dibaca sekarang. Coba lagi sebentar lagi.")
		}
		return
	}

	stores, capped := storesWithin(req.Lat, req.Lng, radiusM, rows, s.cfg.maxResults)
	res = dto.FamilyMartStoresResponseGet{
		Lat:     req.Lat,
		Lng:     req.Lng,
		RadiusM: radiusM,
		Count:   len(stores),
		Capped:  capped,
		Cached:  cached,
		Stores:  stores,
	}
	return
}

// cellRows: the store rows of the ~110 m cell around the point (box of
// radius + pad around the cell centre), from Redis or the database.
func (s *FamilyMartStoreService) cellRows(ctx context.Context, lat float64, lng float64, radiusM int) ([]model.FamilyMartStore, bool, error) {
	cLat := math.Round(lat*1000) / 1000
	cLng := math.Round(lng*1000) / 1000
	fetchR := radiusM + fmStoresPadM
	key := fmt.Sprintf("%s:%s:%.3f,%.3f:%d", fmStoresCacheVersion, s.RepositoryStore.CacheTag(), cLat, cLng, fetchR)

	if s.cfg.cacheTTL > 0 {
		if raw, ok := s.RepositoryCache.GetArea(ctx, key); ok {
			var rows []model.FamilyMartStore
			if json.Unmarshal(raw, &rows) == nil {
				return rows, true, nil
			}
		}
	}

	minLat, maxLat, minLng, maxLng := boundingBox(cLat, cLng, float64(fetchR))
	rows, err := s.RepositoryStore.InBox(ctx, minLat, maxLat, minLng, maxLng, fmStoresSQLCap)
	if err != nil {
		return nil, false, err
	}
	if s.cfg.cacheTTL > 0 {
		if rows == nil {
			rows = []model.FamilyMartStore{}
		}
		if raw, mErr := json.Marshal(rows); mErr == nil {
			s.RepositoryCache.SetArea(ctx, key, raw, s.cfg.cacheTTL)
		}
	}
	return rows, false, nil
}

// storesWithin: the rows within radiusM of the point (exact distance),
// nearest first (ties by store code), at most maxResults; capped = more
// were inside. Rows without coordinates are skipped; the same store code
// twice is kept once.
func storesWithin(lat float64, lng float64, radiusM int, rows []model.FamilyMartStore, maxResults int) ([]dto.FamilyMartStoreItem, bool) {
	out := make([]dto.FamilyMartStoreItem, 0, len(rows))
	seen := map[string]bool{}
	for _, r := range rows {
		if (r.Latitude == 0 && r.Longitude == 0) || math.IsNaN(r.Latitude) || math.IsNaN(r.Longitude) {
			continue
		}
		code := strings.TrimSpace(r.StoreCode)
		if seen[code] {
			continue
		}
		d := haversineM(lat, lng, r.Latitude, r.Longitude)
		if d > float64(radiusM) {
			continue
		}
		seen[code] = true
		item := dto.FamilyMartStoreItem{
			StoreCode: code,
			StoreName: strings.TrimSpace(r.StoreName),
			Lat:       r.Latitude,
			Lng:       r.Longitude,
			DistanceM: int(math.Round(d)),
		}
		if r.Address != nil {
			item.Address = strings.TrimSpace(*r.Address)
		}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DistanceM != out[j].DistanceM {
			return out[i].DistanceM < out[j].DistanceM
		}
		return out[i].StoreCode < out[j].StoreCode
	})
	if maxResults > 0 && len(out) > maxResults {
		return out[:maxResults], true
	}
	return out, false
}
