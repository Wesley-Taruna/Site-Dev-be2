package router

// Side-by-side check, module 3 (site reads): GET /v1/site, /maps,
// /active-list, /:id, /:id/context against site-dev-api 20eeab4.

import (
	"database/sql/driver"
	"net/http"
	"strings"
	"testing"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- copies of site-dev-api 20eeab4 internal/app/dto/site.go (unchanged) ---

type oldSiteMapResp struct {
	StoreCode      string  `json:"store_code"`
	StoreName      string  `json:"store_name"`
	StoreTypeName  string  `json:"store_type_name"`
	SiteType       string  `json:"site_type"`
	Longitude      float64 `json:"longitude"`
	Latitude       float64 `json:"latitude"`
	OpeningStoreDT *string `json:"opening_store_dt"`
}

type oldSiteMapStoreTypeCount struct {
	StoreTypeName string `json:"store_type_name"`
	Total         int    `json:"total"`
}

type oldSiteMapListResp struct {
	Sites      []oldSiteMapResp           `json:"sites"`
	StoreTypes []oldSiteMapStoreTypeCount `json:"store_types"`
}

// ---------------------------------------------------------------------------

func squash(s string) string { return strings.Join(strings.Fields(s), " ") }

func TestSideBySide_SiteDetail(t *testing.T) {
	h, rec, _ := stack(t, detailResults("USER")...)

	code, body := call(t, h, http.MethodGet, "/v1/site/ST1", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, oldSiteST1()), body)
	assert.Equal(t, []string{oldDetailSQL[2]}, rec.sqls)

	code, body = call(t, h, http.MethodGet, "/v1/site/ST404", nil, 1001)
	assert.Equal(t, http.StatusInternalServerError, code)
	assertSameJSON(t, []byte(`{"code":500,"status":"failure","message":"Internal Server Error","errors":{"error":"record not found"}}`), body)
}

func TestSideBySide_SiteIndex(t *testing.T) {
	h, rec, _ := stack(t,
		sqlfake.Result{Match: `count(*)`, Columns: []string{"count"}, Rows: [][]driver.Value{{int64(1)}}},
		sqlfake.Result{Match: `FROM "Sites"`, Columns: siteColumns, Rows: [][]driver.Value{siteST1Row()}},
	)

	code, body := call(t, h, http.MethodGet, "/v1/site?city=Jakarta&region=DKI&sort_by=SiteName1&sort_dir=desc", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldListEnvelope(t, []oldSiteResp{oldSiteST1()}, 1, 1, 0), body)
	assert.Equal(t, []string{
		`SELECT count(*) FROM "Sites" WHERE SiteCity = 'Jakarta' AND SiteRegion = 'DKI'`,
		`SELECT * FROM "Sites" WHERE SiteCity = 'Jakarta' AND SiteRegion = 'DKI' ORDER BY SiteName1 DESC OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY`,
	}, rec.sqls)
}

var storeColumns = []string{"StoreCode", "StoreName", "StoreTypeName", "SiteType", "Longitude", "Latitude", "OpeningStoreDT"}

func TestSideBySide_SiteMaps(t *testing.T) {
	other := siteST1Row()
	other[0], other[1], other[10], other[11] = "ST2", "Toko Baru", -6.3, 106.9
	h, rec, _ := stack(t,
		sqlfake.Result{Match: `AMFIBI_MASTER.dbo.MStore`, Columns: storeColumns[1:6], Rows: [][]driver.Value{
			{"FAMI   KEMANG-raya", "Regular", "Site Active", 106.8, -6.2},
			{"Fami Senayan", "Express", "Site Active", 106.7, -6.1},
			{"Fami Blok M", "Regular", "Site Active", 106.75, -6.24},
		}},
		sqlfake.Result{Match: `FROM "Sites"`, Columns: siteColumns, Rows: [][]driver.Value{
			// "Fami Kemang Raya" is the same store as "FAMI   KEMANG-raya": skipped
			append([]driver.Value{"ST1", "Fami Kemang Raya"}, siteST1Row()[2:]...),
			other,
		}},
	)

	code, body := call(t, h, http.MethodGet, "/v1/site/maps", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, oldSiteMapListResp{
		Sites: []oldSiteMapResp{
			{StoreName: "FAMI   KEMANG-raya", StoreTypeName: "Regular", SiteType: "Site Active", Longitude: 106.8, Latitude: -6.2},
			{StoreName: "Fami Senayan", StoreTypeName: "Express", SiteType: "Site Active", Longitude: 106.7, Latitude: -6.1},
			{StoreName: "Fami Blok M", StoreTypeName: "Regular", SiteType: "Site Active", Longitude: 106.75, Latitude: -6.24},
			{StoreName: "Toko Baru", StoreTypeName: "Requested", SiteType: "Site Requested", Longitude: 106.9, Latitude: -6.3},
		},
		StoreTypes: []oldSiteMapStoreTypeCount{{"Regular", 2}, {"Express", 1}, {"Requested", 1}},
	}), body)
	// site-dev-api also ran `SELECT count(*) FROM "Sites"` (generic FindAll)
	// and ignored the number; not repeated.
	// November scope (API-CHANGES §4): one more query for the request
	// pins' task_status (no row here, so the answer is the old one)
	require.Len(t, rec.sqls, 3)
	assert.Equal(t, squash(`SELECT ms.StoreName, mt.StoreTypeName, 'Site Active' as SiteType, ms.Longitude, ms.Latitude
		FROM AMFIBI_MASTER.dbo.MStore as ms INNER JOIN AMFIBI_MASTER.dbo.MStoreType mt on ms.StoreType = mt.StoreTypeID
		WHERE ms.Longitude != 0 and ms.IsActive = 1 and ms.StoreName != 'FOODTRUCK 1'`), squash(rec.sqls[0]))
	assert.Equal(t, `SELECT * FROM "Sites"`, rec.sqls[1])
	assert.Equal(t, `SELECT SiteID, TaskStatus FROM "Tasks" WHERE SiteID IN ('ST2')`, rec.sqls[2])
}

func TestSideBySide_SiteActiveList(t *testing.T) {
	opened := "2019-05-01"
	h, rec, _ := stack(t,
		sqlfake.Result{Match: `SELECT COUNT(1)`, Columns: []string{"count"}, Rows: [][]driver.Value{{int64(57)}}},
		sqlfake.Result{Match: `AMFIBI_MASTER.dbo.MStore`, Columns: storeColumns, Rows: [][]driver.Value{
			{"F001", "Fami Senayan", "Express", "Site Active", 106.7, -6.1, opened},
			{"F002", "Fami Blok M", "Regular", "Site Active", 106.75, -6.24, nil},
		}},
	)

	code, body := call(t, h, http.MethodGet, "/v1/site/active-list?q=Fami&page=2", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldListEnvelope(t, []oldSiteMapResp{
		{StoreCode: "F001", StoreName: "Fami Senayan", StoreTypeName: "Express", SiteType: "Site Active", Longitude: 106.7, Latitude: -6.1, OpeningStoreDT: &opened},
		{StoreCode: "F002", StoreName: "Fami Blok M", StoreTypeName: "Regular", SiteType: "Site Active", Longitude: 106.75, Latitude: -6.24},
	}, 57, 2, 0), body)
	require.Len(t, rec.sqls, 2)
	assert.Equal(t, squash(`SELECT COUNT(1) FROM AMFIBI_MASTER.dbo.MStore as ms INNER JOIN AMFIBI_MASTER.dbo.MStoreType mt on ms.StoreType = mt.StoreTypeID
		WHERE ms.Longitude != 0 and ms.IsActive = 1 and ms.StoreName != 'FOODTRUCK 1' AND (ms.StoreName LIKE '%Fami%' OR mt.StoreTypeName LIKE '%Fami%')`), squash(rec.sqls[0]))
	assert.True(t, strings.HasSuffix(squash(rec.sqls[1]), `ORDER BY ms.StoreCode OFFSET 20 ROWS FETCH NEXT 20 ROWS ONLY`), rec.sqls[1])
}

func TestSideBySide_SiteContext(t *testing.T) {
	h, rec, _ := stack(t, detailResults("USER")...)

	code, body := call(t, h, http.MethodGet, "/v1/site/ST1/context", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, *oldDetailST1().Context), body)
	assert.Equal(t, oldDetailSQL[5:8], rec.sqls)

	// never filled in: site-dev-api answers 500 "record not found"
	h, _, _ = stack(t, noRows(`FROM "SiteContexts"`))
	code, body = call(t, h, http.MethodGet, "/v1/site/ST1/context", nil, 1001)
	assert.Equal(t, http.StatusInternalServerError, code)
	assertSameJSON(t, []byte(`{"code":500,"status":"failure","message":"Internal Server Error","errors":{"error":"record not found"}}`), body)
}
