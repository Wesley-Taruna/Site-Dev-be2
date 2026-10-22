package router

// November scope, API-CHANGES §11 (+ 20/21 Oct): the web portal endpoints.
// New; shapes and rules from site-dev-mobile src/lib/types.ts "Portal web"
// and mock.ts (portalGroup, portalRow, updateSiteAds).

import (
	"database/sql/driver"
	"net/http"
	"testing"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func portalListResults() []sqlfake.Result {
	return []sqlfake.Result{
		userRow(4001, "putri", "", "ADMIN"),
		userRow(1001, "andi", "", "USER"),
		{Match: `SELECT count(*) FROM "Tasks" JOIN Sites`, Columns: []string{"count"}, Rows: [][]driver.Value{{int64(2)}}},
		{Match: `SELECT Tasks.* FROM "Tasks" JOIN Sites`, Columns: taskColumnsNov, Rows: [][]driver.Value{
			{"TS1", "ST1", "HOLD_DIV_HEAD", int64(1001), int64(3002), created, acted, int64(1)},
			{"TS2", "ST2", "COMPLETED", int64(1001), int64(3002), created, updated, int64(1)},
		}},
		{Match: `FROM "Sites" WHERE SiteID IN`, Columns: siteColumns, Rows: [][]driver.Value{siteST1Row(), append([]driver.Value{"ST2", "Toko Dua"}, siteST1Row()[2:]...)}},
		{Match: `MIN(SubmittedAt)`, Columns: []string{"TaskID", "SubmittedAt"}, Rows: [][]driver.Value{{"TS1", updated}, {"TS2", created}}},
		{Match: `SELECT ApprovalHDR.TaskID, ApprovalDET.Notes`, Columns: []string{"TaskID", "Notes"}, Rows: [][]driver.Value{{"TS1", "tunggu izin"}}},
		{Match: `FROM "SiteAds"`, Columns: []string{"SiteID", "AdsValue", "AdsNote", "UpdatedBy", "UpdatedAt"}, Rows: [][]driver.Value{{"ST2", int64(9400000), "", int64(4001), acted}}},
	}
}

func TestPortal_List(t *testing.T) {
	h, rec, _ := stack(t, portalListResults()...)

	code, body := call(t, h, http.MethodGet, "/v1/portal/site?q=toko&page=1&limit=5", nil, 4001)

	require.Equal(t, http.StatusOK, code, string(body))
	var rows []map[string]any
	require.NoError(t, jsonUnmarshalData(body, &rows))
	require.Len(t, rows, 2)
	assert.Equal(t, "HOLD", rows[0]["status_group"])
	assert.Equal(t, "tunggu izin", rows[0]["hold_note"])
	assert.Equal(t, "andi", rows[0]["created_by_name"])
	assert.Equal(t, "1001", rows[0]["created_by_nik"])
	assert.Equal(t, "2026-10-03T14:15:00Z", rows[0]["submitted_at"])
	assert.Nil(t, rows[0]["ads"])
	assert.Equal(t, "APPROVED", rows[1]["status_group"])
	assert.NotContains(t, rows[1], "hold_note")
	assert.Equal(t, map[string]any{"ads_value": float64(9400000), "ads_note": "", "updated_by_name": "putri", "updated_at": "2026-10-04T10:30:00Z"}, rows[1]["ads"])
	assert.Contains(t, string(body), `"total":2,"page":1,"per_page":5`)

	all := `('WAITING_DEPT_HEAD','WAITING_DIV_HEAD','HOLD_DEPT_HEAD','HOLD_DIV_HEAD','COMPLETED','APPROVED','REJECTED','REVISION')`
	assert.Equal(t, `SELECT count(*) FROM "Tasks" JOIN Sites ON Sites.SiteID = Tasks.SiteID WHERE Tasks.TaskStatus IN `+all+` AND (Sites.SiteName1 LIKE '%toko%' OR Sites.SiteID LIKE '%toko%')`, rec.sqls[1])
	assert.Equal(t, `SELECT Tasks.* FROM "Tasks" JOIN Sites ON Sites.SiteID = Tasks.SiteID WHERE Tasks.TaskStatus IN `+all+` AND (Sites.SiteName1 LIKE '%toko%' OR Sites.SiteID LIKE '%toko%') ORDER BY Tasks.UpdatedAt DESC, Tasks.TaskID DESC OFFSET 0 ROW FETCH NEXT 5 ROWS ONLY`, rec.sqls[2])
}

func TestPortal_ListGroupAndAccess(t *testing.T) {
	h, rec, _ := stack(t, portalListResults()...)

	code, _ := call(t, h, http.MethodGet, "/v1/portal/site?status=REJECTED", nil, 4001)
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, rec.sqls[1], `WHERE Tasks.TaskStatus IN ('REJECTED','REVISION')`)

	code, body := call(t, h, http.MethodGet, "/v1/portal/site?status=DRAFT", nil, 4001)
	assert.Equal(t, http.StatusBadRequest, code, string(body))

	code, body = call(t, h, http.MethodGet, "/v1/portal/site", nil, 1001)
	assert.Equal(t, http.StatusForbidden, code, "default: site staff do not use the portal")
	assert.Contains(t, string(body), "Portal Site Dev hanya untuk pengguna kantor")
}

func TestPortal_StatusCount(t *testing.T) {
	h, _, _ := stack(t,
		userRow(3002, "dodi", "", "DIV_HEAD"),
		sqlfake.Result{Match: `SELECT TaskStatus, COUNT(1)`, Columns: []string{"TaskStatus", "Count"}, Rows: [][]driver.Value{
			{"DRAFT", int64(9)}, {"WAITING_DEPT_HEAD", int64(3)}, {"HOLD_DEPT_HEAD", int64(1)}, {"HOLD_DIV_HEAD", int64(2)},
			{"COMPLETED", int64(5)}, {"REVISION", int64(4)}, {"REJECTED", int64(1)},
		}},
	)

	code, body := call(t, h, http.MethodGet, "/v1/portal/site/status/count", nil, 3002)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success","data":[
		{"status_group":"WAITING_DEPT_HEAD","count":3},{"status_group":"WAITING_DIV_HEAD","count":0},{"status_group":"HOLD","count":3},
		{"status_group":"APPROVED","count":5},{"status_group":"REJECTED","count":5}]}`), body)
}

func TestPortal_Report(t *testing.T) {
	results := append([]sqlfake.Result{
		userRow(4001, "putri", "", "ADMIN"),
		{Match: `FROM "TaskHistories"`, Columns: histCol, Rows: [][]driver.Value{
			{int64(1), "TS1", "CREATE", int64(1001), "", created, "USER", int64(1)},
			{int64(2), "TS1", "SUBMIT", int64(1001), "Approval dikirim ke Dept Head, status PENDING (catatan: Pemilik minta sewa 5 tahun)", updated, "USER", int64(1)},
		}},
		noRows(`FROM "SiteAIInsights"`),
		{Match: `FROM "SiteAds"`, Columns: []string{"SiteID", "AdsValue", "AdsNote", "UpdatedBy", "UpdatedAt"}, Rows: [][]driver.Value{{"ST1", nil, "belum", int64(4001), acted}}},
	}, detailResults("USER")...)
	h, _, _ := stack(t, results...)

	code, body := call(t, h, http.MethodGet, "/v1/portal/site/ST1/report", nil, 4001)

	require.Equal(t, http.StatusOK, code, string(body))
	var res map[string]any
	require.NoError(t, jsonUnmarshalData(body, &res))
	site := res["site"].(map[string]any)
	assert.Equal(t, "ST1", site["site_id"])
	assert.Equal(t, "andi", site["created_by_name"])
	assert.Equal(t, "1001", site["created_by_nik"])
	assert.Equal(t, "WAITING_DIV_HEAD", res["status_group"])
	assert.Equal(t, "Pemilik minta sewa 5 tahun", res["staff_note"])
	assert.Nil(t, res["insight"], "never generated")
	assert.NotNil(t, res["context"])
	assert.Len(t, res["task"].(map[string]any)["approval"], 2)
	assert.Equal(t, map[string]any{"ads_value": nil, "ads_note": "belum", "updated_by_name": "putri", "updated_at": "2026-10-04T10:30:00Z"}, res["ads"])
}

func TestStaffNoteOf(t *testing.T) {
	// read through the report: the automatic text is not a staff note
	results := append([]sqlfake.Result{
		userRow(4001, "putri", "", "ADMIN"),
		{Match: `FROM "TaskHistories"`, Columns: histCol, Rows: [][]driver.Value{
			{int64(2), "TS1", "RESUBMIT", int64(1001), "Approval dikirim ke Dept Head, status PENDING (catatan: Site diajukan oleh andi untuk proses approval)", updated, "USER", int64(2)},
		}},
		noRows(`FROM "SiteAIInsights"`),
	}, detailResults("USER")...)
	h, _, _ := stack(t, results...)
	code, body := call(t, h, http.MethodGet, "/v1/portal/site/ST1/report", nil, 4001)
	require.Equal(t, http.StatusOK, code, string(body))
	assert.Contains(t, string(body), `"staff_note":""`)
}

func TestPortal_UpdateAds(t *testing.T) {
	results := []sqlfake.Result{
		userRow(4001, "putri", "", "ADMIN"),
		userRow(3001, "citra", "", "DEPT_HEAD"),
		{Match: `UPDATE "SiteAds"`, Affected: sqlfake.Int64(0)},
		{Match: `FROM "Tasks" WHERE SiteID = @p1 ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY |ST1`, Columns: taskColumns,
			Rows: [][]driver.Value{{"TS1", "ST1", "COMPLETED", int64(1001), int64(3002), created, updated}}},
		{Match: `FROM "Tasks" WHERE SiteID = @p1 ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY |ST9`, Columns: taskColumns,
			Rows: [][]driver.Value{{"TS9", "ST9", "DRAFT", int64(1001), int64(1001), created, updated}}},
	}
	h, rec, _ := stack(t, results...)

	code, body := call(t, h, http.MethodPut, "/v1/portal/site/ST1/ads", map[string]any{"ads_value": 11850000, "ads_note": "  sewa naik  "}, 4001)

	require.Equal(t, http.StatusOK, code, string(body))
	var res map[string]any
	require.NoError(t, jsonUnmarshalData(body, &res))
	assert.Equal(t, float64(11850000), res["ads_value"])
	assert.Equal(t, "sewa naik", res["ads_note"])
	assert.Equal(t, "putri", res["updated_by_name"])
	sqls := stableSQL(rec.sqls)
	assert.Contains(t, sqls, `UPDATE "SiteAds" SET "AdsNote"='sewa naik',"AdsValue"=11850000,"UpdatedAt"='<time>',"UpdatedBy"=4001 WHERE SiteID = 'ST1'`)
	assert.Contains(t, sqls, `INSERT INTO "SiteAds" ("SiteID","AdsValue","AdsNote","UpdatedBy","UpdatedAt") VALUES ('ST1',11850000,'sewa naik',4001,'<time>');`)

	for name, tc := range map[string]struct {
		site string
		body map[string]any
		user int64
		code int
		want string
	}{
		"not admin":     {"ST1", map[string]any{"ads_value": 1}, 3001, http.StatusForbidden, "hanya admin"},
		"negative":      {"ST1", map[string]any{"ads_value": -1}, 4001, http.StatusUnprocessableEntity, `"ads_value":"The ads value must be between 0 and 10000000000."`},
		"not whole":     {"ST1", map[string]any{"ads_value": 1.5}, 4001, http.StatusUnprocessableEntity, ""},
		"not in portal": {"ST9", map[string]any{"ads_value": nil}, 4001, http.StatusBadRequest, "ADS hanya untuk site yang tampil di portal."},
		"note too long": {"ST1", map[string]any{"ads_note": string(make([]byte, 301))}, 4001, http.StatusUnprocessableEntity, `"ads_note"`},
	} {
		code, body := call(t, h, http.MethodPut, "/v1/portal/site/"+tc.site+"/ads", tc.body, tc.user)
		assert.Equal(t, tc.code, code, name+": "+string(body))
		assert.Contains(t, string(body), tc.want, name)
	}
}
