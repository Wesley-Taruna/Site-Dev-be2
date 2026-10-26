package router

// Check of the rewritten local §8 (site-dev-api local commit 93633f0):
// GET /v1/task/nearby and the duplicate check on POST /v1/task/SUBMITED.
// JSON shape is the local commit's NearbyRequestResp; statuses without §5.

import (
	"database/sql/driver"
	"net/http"
	"testing"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var anyArg = mock.Anything

// copy of site-dev-api 93633f0 internal/app/dto/task.go NearbyRequestResp
type oldNearbyRequestResp struct {
	SiteID     string  `json:"site_id"`
	SiteName   string  `json:"site_name"`
	TaskStatus string  `json:"task_status"`
	OwnerID    int64   `json:"owner_id"`
	OwnerName  string  `json:"owner_name"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	DistanceM  int     `json:"distance_m"`
}

const nLat, nLng = -6.2245, 106.8095

var pointColumns = []string{"SiteID", "SiteName1", "SiteLat", "SiteLon", "TaskStatus", "CreatedBy"}

func nearbyPoints() sqlfake.Result {
	return sqlfake.Result{Match: `JOIN Tasks ON Tasks.SiteID = Sites.SiteID`, Columns: pointColumns, Rows: [][]driver.Value{
		{"ST7", "Toko Lain", nLat + 0.001, nLng, "WAITING_DIV_HEAD", int64(5005)},
		{"ST8", "Draf Lain", nLat, nLng, "DRAFT", int64(5005)},
	}}
}

func TestNearbyRequests(t *testing.T) {
	h, rec, _ := stack(t, nearbyPoints(), userRow(5005, "eko", "", "USER"))

	code, body := call(t, h, http.MethodGet, "/v1/task/nearby?lat=-6.2245&lng=106.8095&exclude_site_id=ST1", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, []oldNearbyRequestResp{{SiteID: "ST7", SiteName: "Toko Lain", TaskStatus: "WAITING_DIV_HEAD", OwnerID: 5005, OwnerName: "eko", Lat: nLat + 0.001, Lng: nLng, DistanceM: 111}}), body)
	assert.Regexp(t, `^SELECT Sites.SiteID, Sites.SiteName1, Sites.SiteLat, Sites.SiteLon, Tasks.TaskStatus, Tasks.CreatedBy FROM "Sites" JOIN Tasks ON Tasks.SiteID = Sites.SiteID WHERE \(Sites.SiteLat BETWEEN -6\.22\d+ AND -6\.22\d+ AND Sites.SiteLon BETWEEN 106\.80\d+ AND 106\.81\d+\) AND \(\(Tasks.TaskStatus LIKE 'WAITING\[_\]%' OR Tasks.TaskStatus IN \('REVISION','COMPLETED','HOLD_DEPT_HEAD','HOLD_DIV_HEAD'\)\)\)$`, rec.sqls[0])

	code, _ = call(t, h, http.MethodGet, "/v1/task/nearby?lat=-6.2&lng=106.8&radius=3000", nil, 1001)
	assert.Equal(t, http.StatusBadRequest, code)
	code, _ = call(t, h, http.MethodGet, "/v1/task/nearby?lat=x&lng=1", nil, 1001)
	assert.Equal(t, http.StatusUnprocessableEntity, code)
}

// draftTS1 answers the submit's task lookup with a DRAFT task.
var draftTS1 = sqlfake.Result{Match: `FROM "Tasks" WHERE TaskID = @p1`, Columns: taskColumns, Rows: [][]driver.Value{{"TS1", "ST1", "DRAFT", int64(1001), int64(1001), created, updated}}}

func duplicateCheckOn(t *testing.T) {
	testEnv.Set("nearby_request.duplicate_check", "confirm")
	t.Cleanup(func() { testEnv.Set("nearby_request.duplicate_check", "") })
}

func TestDuplicateCheck_Blocks(t *testing.T) {
	duplicateCheckOn(t)
	withFlow(t)
	results := append([]sqlfake.Result{nearbyPoints(), userRow(5005, "eko", "", "USER")}, saveResults("DRAFT")...)
	h, _, fdb := stack(t, results...)

	code, body := call(t, h, http.MethodPost, "/v1/task/SUBMITED", map[string]any{"site": map[string]any{"site_id": "ST1", "site_name_1": "Toko Kemang", "site_lat": nLat, "site_lon": nLng, "site_sales_area": 120.0}}, 1001)

	assert.Equal(t, http.StatusConflict, code)
	assertSameJSON(t, []byte(`{"code":409,"status":"failure","message":"Conflict","data":[{"site_id":"ST7","site_name":"Toko Lain","task_status":"WAITING_DIV_HEAD","owner_id":5005,"owner_name":"eko","lat":-6.2235,"lng":106.8095,"distance_m":111}],"errors":{"error":"DUPLICATE_NEARBY: sudah ada 1 pengajuan aktif dalam 200 m dari titik ini. Cek dulu, lalu kirim ulang dengan confirm_duplicate = true bila tetap ingin mengirim."}}`), body)
	assert.NotContains(t, fdb.Queries(), "BEGIN", "a 409 saves nothing")
}

func TestDuplicateCheck_ConfirmedSendsWithHistory(t *testing.T) {
	duplicateCheckOn(t)
	m := withFlow(t)
	m.mail.On("SendApprovalPending", anyArg, anyArg, anyArg, anyArg).Return(nil).Maybe()
	results := append([]sqlfake.Result{draftTS1, nearbyPoints(), userRow(5005, "eko", "", "USER"), chainResult(),
		{Match: `INSERT INTO "ApprovalDET"`, Columns: []string{"ID"}, Rows: [][]driver.Value{{int64(31)}, {int64(32)}}}}, saveResults("DRAFT")...)
	h, rec, _ := stack(t, results...)

	code, body := call(t, h, http.MethodPost, "/v1/task/SUBMITED", map[string]any{"site": map[string]any{"site_id": "ST1", "site_name_1": "Toko Kemang", "site_lat": nLat, "site_lon": nLng, "site_sales_area": 120.0}, "confirm_duplicate": true}, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Contains(t, stableSQL(rec.sqls), `INSERT INTO "TaskHistories" ("TaskID","Action","ActorID","Notes","CreatedAt","ActorRole","Round") OUTPUT INSERTED."ID" VALUES ('TS1','DUPLICATE_CONFIRMED',1001,'Staf tetap mengirim walau ada pengajuan aktif lain dalam 200 m: ST7 Toko Lain (WAITING_DIV_HEAD, 111 m)','<time>','USER',1);`)
}

func TestDuplicateCheck_OffByDefault(t *testing.T) {
	m := withFlow(t)
	m.mail.On("SendApprovalPending", anyArg, anyArg, anyArg, anyArg).Return(nil).Maybe()
	results := append([]sqlfake.Result{draftTS1, nearbyPoints(), chainResult()}, saveResults("DRAFT")...)
	h, rec, _ := stack(t, results...)

	code, body := call(t, h, http.MethodPost, "/v1/task/SUBMITED", map[string]any{"site": map[string]any{"site_id": "ST1", "site_name_1": "Toko Kemang", "site_lat": nLat, "site_lon": nLng, "site_sales_area": 120.0}}, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	for _, s := range rec.sqls {
		assert.NotContains(t, s, "JOIN Tasks ON Tasks.SiteID = Sites.SiteID", "no nearby query when the check is off")
	}
}
