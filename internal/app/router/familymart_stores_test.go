package router

// GET /v1/site/familymart-stores (Wes, 29 Oct): the active FamilyMart
// stores within 1 km of a point, nearest first, from the store database
// (default: the main connection, AMFIBI_MASTER.dbo.MStore).

import (
	"database/sql/driver"
	"net/http"
	"strings"
	"testing"
	"time"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fmStoreBox = sqlfake.Result{
	Match:   `SELECT TOP (2000) StoreCode AS StoreCode, StoreName AS StoreName, Latitude AS Latitude, Longitude AS Longitude FROM AMFIBI_MASTER.dbo.MStore WHERE Latitude BETWEEN @p1 AND @p2 AND Longitude BETWEEN @p3 AND @p4 AND StoreName <> 'FOODTRUCK 1' AND IsActive = 1`,
	Columns: []string{"StoreCode", "StoreName", "Latitude", "Longitude"},
	Rows: [][]driver.Value{
		{"UJI003", "FM Uji Kuningan", -6.22410, 106.83140},
		{"UJI020", "FM Uji Setiabudi", -6.20900, 106.82980},
		{"UJI018", "FM Uji Gatot Subroto", -6.23240, 106.82160},
	},
}

func TestFamilyMartStores(t *testing.T) {
	redis := withRedis(t)
	h, rec, _ := stack(t, fmStoreBox)

	code, body := call(t, h, http.MethodGet, "/v1/site/familymart-stores?lat=-6.2160&lng=106.8300", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success","data":{
		"lat":-6.216,"lng":106.83,"radius_m":1000,"count":2,"capped":false,"cached":false,
		"stores":[
			{"store_code":"UJI020","store_name":"FM Uji Setiabudi","address":"","lat":-6.209,"lng":106.8298,"distance_m":779},
			{"store_code":"UJI003","store_name":"FM Uji Kuningan","address":"","lat":-6.2241,"lng":106.8314,"distance_m":914}
		]}}`), body)
	assert.Equal(t, 1, countSQL(rec, "AMFIBI_MASTER.dbo.MStore"))
	var cacheKey string
	for _, k := range redis.Keys() {
		if strings.Contains(k, "cache:nearby_area:fmstores:v1:") {
			cacheKey = k
		}
	}
	require.NotEmpty(t, cacheKey, "rows cached in Redis")
	assert.Equal(t, time.Hour, redis.TTL(cacheKey))

	// same ~110 m cell: from Redis, no second query
	code, body = call(t, h, http.MethodGet, "/v1/site/familymart-stores?lat=-6.2161&lng=106.8301", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assert.Contains(t, string(body), `"cached":true`)
	assert.Equal(t, 1, countSQL(rec, "AMFIBI_MASTER.dbo.MStore"))
}

func TestFamilyMartStores_Radius(t *testing.T) {
	h, _, _ := stack(t, fmStoreBox)

	code, body := call(t, h, http.MethodGet, "/v1/site/familymart-stores?lat=-6.2160&lng=106.8300&radius=800", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Contains(t, string(body), `"radius_m":800,"count":1,`)
	assert.Contains(t, string(body), `"store_code":"UJI020"`)
}

func TestFamilyMartStores_Validation(t *testing.T) {
	h, _, _ := stack(t, fmStoreBox)
	for _, tc := range []struct {
		name string
		path string
		user int64
		code int
		has  string
	}{
		{"no token", "/v1/site/familymart-stores?lat=-6.216&lng=106.83", 0, http.StatusForbidden, ""},
		{"lat missing", "/v1/site/familymart-stores?lng=106.83", 1001, http.StatusUnprocessableEntity, "lat"},
		{"lng not a number", "/v1/site/familymart-stores?lat=-6.216&lng=abc", 1001, http.StatusUnprocessableEntity, "lng"},
		{"radius not a number", "/v1/site/familymart-stores?lat=-6.216&lng=106.83&radius=1km", 1001, http.StatusUnprocessableEntity, "radius"},
		{"outside Indonesia", "/v1/site/familymart-stores?lat=51.5&lng=-0.12", 1001, http.StatusBadRequest, "Koordinat tidak valid (harus di Indonesia)."},
		{"radius over 1000", "/v1/site/familymart-stores?lat=-6.216&lng=106.83&radius=1500", 1001, http.StatusBadRequest, "radius maksimal 1000 m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, body := call(t, h, http.MethodGet, tc.path, nil, tc.user)
			assert.Equal(t, tc.code, code, string(body))
			assert.True(t, strings.Contains(string(body), tc.has), string(body))
		})
	}
}

func TestFamilyMartStores_BadStoreConfigIs503(t *testing.T) {
	testEnv.Set("store_database.table", "MStore; DROP TABLE Sites")
	t.Cleanup(func() { testEnv.Set("store_database.table", "") })
	h, rec, _ := stack(t, fmStoreBox)

	code, body := call(t, h, http.MethodGet, "/v1/site/familymart-stores?lat=-6.216&lng=106.83", nil, 1001)

	assert.Equal(t, http.StatusServiceUnavailable, code)
	assertSameJSON(t, []byte(`{"code":503,"status":"failure","message":"Service Unavailable","errors":{"error":"Data toko FamilyMart belum diatur dengan benar di server ini ([store_database])."}}`), body)
	assert.Equal(t, 0, countSQL(rec, "MStore"))
}

// countSQL: how many logged statements contain part.
func countSQL(l *sqlLog, part string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, s := range l.sqls {
		if strings.Contains(s, part) {
			n++
		}
	}
	return n
}
