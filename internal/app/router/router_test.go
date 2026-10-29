package router

// Side-by-side check for the task slice: the whole new stack (Router ->
// Authorized -> TaskHandler -> TaskService -> repositories -> gorm SQL Server
// dialect) runs against an in-memory database (sqlfake). The expected body is
// built with copies of site-dev-api's own response types (20eeab4,
// internal/app/dto/task.go) and the same familabs envelope, so a field
// rename, a missing field or a changed envelope fails here.

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/fami-app-v2/familib/familabs"
	jwtx "gitlab.com/fami-app-v2/familib/familabs/jwt"
	"gorm.io/gorm/logger"
)

// --- copies of site-dev-api 20eeab4 internal/app/dto/task.go (unchanged) ---

type oldTaskResp struct {
	TaskID       string    `json:"task_id"`
	SiteID       string    `json:"site_id"`
	SiteName1    string    `json:"site_name_1"`
	SiteAddress1 string    `json:"site_address_1"`
	SiteAddress2 string    `json:"site_address_2"`
	SiteAddress3 string    `json:"site_address_3"`
	TaskStatus   string    `json:"task_status"`
	CreatedBy    int64     `json:"created_by"`
	UpdatedBy    int64     `json:"updated_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type oldTaskHistoryResp struct {
	ID        uint      `json:"id"`
	Action    string    `json:"action"`
	ActorID   int64     `json:"actor_id"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

// ---------------------------------------------------------------------------

const testJwtKey = "unit-test-signing-key" // test-only

type sqlLog struct {
	mu   sync.Mutex
	sqls []string
}

func (l *sqlLog) LogMode(logger.LogLevel) logger.Interface      { return l }
func (l *sqlLog) Info(context.Context, string, ...interface{})  {}
func (l *sqlLog) Warn(context.Context, string, ...interface{})  {}
func (l *sqlLog) Error(context.Context, string, ...interface{}) {}
func (l *sqlLog) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	s, _ := fc()
	l.mu.Lock()
	l.sqls = append(l.sqls, s)
	l.mu.Unlock()
}

var (
	created = time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	updated = time.Date(2026, 10, 3, 14, 15, 0, 0, time.UTC)
)

func setup(t *testing.T, role string) (*echo.Echo, *sqlLog) {
	conn, err := sqlfake.Open(&sqlfake.DB{Results: []sqlfake.Result{
		{Match: `FROM "Users"`, Columns: []string{"UserID", "Username", "Email", "RoleID", "CreatedAt", "UpdatedAt"},
			Rows: [][]driver.Value{{int64(1001), "andi", "", role, created, created}}},
		{Match: `count(*)`, Columns: []string{"count"}, Rows: [][]driver.Value{{int64(2)}}},
		{Match: `|ST404`}, // no task for this site
		{Match: `WHERE SiteID = @p1 ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY |ST1`, Columns: []string{"TaskID", "SiteID", "TaskStatus", "CreatedBy", "UpdatedBy", "CreatedAt", "UpdatedAt"},
			Rows: [][]driver.Value{{"TS1", "ST1", "WAITING_DEPT_HEAD", int64(1001), int64(1001), created, updated}}},
		{Match: `FROM "Tasks"`, Columns: []string{"TaskID", "SiteID", "TaskStatus", "CreatedBy", "UpdatedBy", "CreatedAt", "UpdatedAt"},
			Rows: [][]driver.Value{
				{"TS1", "ST1", "WAITING_DEPT_HEAD", int64(1001), int64(1001), created, updated},
				{"TS2", "ST2", "SUBMITED", int64(1001), int64(0), created, created},
			}},
		{Match: `FROM "Sites"`, Columns: []string{"SiteID", "SiteName1", "SiteAddress1", "SiteAddress2", "SiteAddress3", "CreatedBy", "CreatedAt"},
			Rows: [][]driver.Value{{"ST1", "Toko Kemang", "Jl. Kemang Raya 1", "", "Jakarta Selatan", int64(1001), created}}},
		{Match: `FROM "TaskHistories"`, Columns: []string{"ID", "TaskID", "Action", "ActorID", "Notes", "CreatedAt"},
			Rows: [][]driver.Value{
				{int64(1), "TS1", "CREATE", int64(1001), "", created},
				{int64(2), "TS1", "SUBMIT", int64(1001), "Site dikirim", updated},
			}},
	}})
	require.NoError(t, err)

	e, rec, _ := stackOn(t, conn)
	return e, rec
}

func get(t *testing.T, h http.Handler, path string) (int, []byte) {
	token, err := jwtx.NewJWT([]byte(testJwtKey)).Create(&jwtx.UserClaim{UserID: 1001, ExpiresAt: time.Now().Add(time.Hour).Unix()})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code, rr.Body.Bytes()
}

// oldEnvelope builds what site-dev-api's handler sends for the same data:
// ctx.DataList(resp, total, paginator) / ctx.Data(resp) + ctx.Serve(nil).
func oldListEnvelope(t *testing.T, data any, total int64, page, perPage int) []byte {
	r := familabs.NewResponse()
	r.SetDataList(data, total, &familabs.Paginator{Page: page, PerPage: perPage})
	b, err := json.Marshal(r)
	require.NoError(t, err)
	return b
}

func oldDataEnvelope(t *testing.T, data any) []byte {
	r := familabs.NewResponse()
	r.SetData(data)
	b, err := json.Marshal(r)
	require.NoError(t, err)
	return b
}

func assertSameJSON(t *testing.T, want, got []byte) {
	var w, g any
	require.NoError(t, json.Unmarshal(want, &w))
	require.NoError(t, json.Unmarshal(got, &g), string(got))
	assert.Equal(t, w, g, "want %s\n got %s", want, got)
	// byte-identical too (same field order), not only the same content
	var wb, gb bytes.Buffer
	require.NoError(t, json.Compact(&wb, want))
	require.NoError(t, json.Compact(&gb, got))
	assert.Equal(t, wb.String(), gb.String())
}

// siteDevAPIRoutes is every route of site-dev-api 20eeab4 (all handlers'
// URLMapping under "/v1/"), 47 in total.
var siteDevAPIRoutes = []string{
	"DELETE /v1/users/:id",
	"GET /v1/approval-matrix",
	"GET /v1/attachment/:id/file",
	"GET /v1/dashboard/task/history",
	"GET /v1/dashboard/task/status/count",
	"GET /v1/geocoding/reverse",
	"GET /v1/geocoding/search",
	"GET /v1/health-check",
	"GET /v1/roles",
	"GET /v1/settings/limits",
	"GET /v1/site",
	"GET /v1/site/:id",
	"GET /v1/site/:id/context",
	"GET /v1/site/active-list",
	"GET /v1/site/maps",
	"GET /v1/task",
	"GET /v1/task/:id",
	"GET /v1/task/:id/attachment",
	"GET /v1/task/:id/history",
	"GET /v1/task/ai-insight/:site_id",
	"GET /v1/task/ai-insight/quota",
	"GET /v1/task/approval-history",
	"GET /v1/task/pending-approvals",
	"GET /v1/task/site-list",
	"GET /v1/task/status/count",
	"GET /v1/users",
	"GET /v1/wilayah/provinces",
	"GET /v1/wilayah/provinces/:code",
	"GET /v1/wilayah/search",
	"POST /v1/approval-matrix",
	"POST /v1/queue/ai-insight",
	"POST /v1/queue/ai-insight-quota-flush",
	"POST /v1/queue/approval-email",
	"POST /v1/settings/quota-flush",
	"POST /v1/task/:id/approve",
	"POST /v1/task/:status",
	"POST /v1/task/ai-insight",
	"POST /v1/task/ai-insight/preview",
	"POST /v1/upload/add",
	"POST /v1/upload/remove",
	"POST /v1/users",
	"POST /v1/validation",
	"POST /v1/validation/approval",
	"PUT /v1/approval-matrix/:id",
	"PUT /v1/settings/limits",
	"PUT /v1/task/context/:site_id",
	"PUT /v1/users/:id",
}

// extraRoutes are routes the new service has that site-dev-api 20eeab4 does
// not: the boilerplate sample (kept until decided) and later the local and
// November additions.
var extraRoutes = map[string]bool{
	"GET /v1/person":        true,
	"GET /v1/person/:id":    true,
	"POST /v1/person":       true,
	"PUT /v1/person/:id":    true,
	"DELETE /v1/person/:id": true,
	// local §8 (93633f0), rewritten
	"GET /v1/task/nearby": true,
	// local §6 (2191e07), rewritten
	"GET /v1/site/nearby-places": true,
	// November scope (review §3a #32-#41)
	"GET /v1/activity":                 true,
	"GET /v1/activity/unread-count":    true,
	"POST /v1/activity/seen":           true,
	"GET /v1/task/approver-news":       true,
	"POST /v1/task/approver-news/seen": true,
	"GET /v1/competitor-brands":        true,
	"GET /v1/portal/site":              true,
	"GET /v1/portal/site/status/count": true,
	"GET /v1/portal/site/:id/report":   true,
	"PUT /v1/portal/site/:id/ads":      true,
	// Wes, 26 Oct: staff delete a never-sent draft / cancel a returned request
	"DELETE /v1/site/:id":      true,
	"POST /v1/task/:id/cancel": true,
	// Wes, 29 Oct: FamilyMart stores within 1 km for the map
	"GET /v1/site/familymart-stores": true,
}

// TestRoutesMatchSiteDevAPI: the boilerplate's group "v1/" + handler key ends
// up at exactly site-dev-api's paths; nothing outside the list exists.
func TestRoutesMatchSiteDevAPI(t *testing.T) {
	require.Len(t, siteDevAPIRoutes, 47)
	e, _ := setup(t, "USER")
	known := map[string]bool{}
	for _, r := range siteDevAPIRoutes {
		known[r] = true
	}
	ported := map[string]bool{}
	for _, r := range e.Routes() {
		if !strings.HasPrefix(r.Path, "/v1/") || r.Method == "echo_route_not_found" {
			continue
		}
		route := r.Method + " " + r.Path
		if extraRoutes[route] {
			continue
		}
		assert.True(t, known[route], "route %s is not in site-dev-api", route)
		ported[route] = true
	}
	var missing []string
	for _, r := range siteDevAPIRoutes {
		if !ported[r] {
			missing = append(missing, r)
		}
	}
	sort.Strings(missing)
	assert.Empty(t, missing, "every site-dev-api route exists")
	assert.Subset(t, siteDevAPIRoutes, keys(ported))
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestSideBySide_TaskList_Staff(t *testing.T) {
	h, rec := setup(t, "USER")

	code, body := get(t, h, "/v1/task?status=WAITING_DEPT_HEAD")

	require.Equal(t, http.StatusOK, code, string(body))
	want := oldListEnvelope(t, []oldTaskResp{
		{TaskID: "TS1", SiteID: "ST1", SiteName1: "Toko Kemang", SiteAddress1: "Jl. Kemang Raya 1", SiteAddress3: "Jakarta Selatan",
			TaskStatus: "WAITING_DEPT_HEAD", CreatedBy: 1001, UpdatedBy: 1001, CreatedAt: created, UpdatedAt: updated},
		{TaskID: "TS2", SiteID: "ST2", TaskStatus: "SUBMITED", CreatedBy: 1001, CreatedAt: created, UpdatedAt: created},
	}, 2, 0, 0)
	assertSameJSON(t, want, body)

	// SQL site-dev-api runs for the same request (read from its
	// middleware.Authenticated user lookup, TaskService.ListTasks,
	// baseFindAll and SiteRepository.FindByIDs)
	assert.Equal(t, []string{
		`SELECT * FROM "Users" WHERE UserID = 1001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT count(*) FROM "Tasks" WHERE TaskStatus = 'WAITING_DEPT_HEAD' AND CreatedBy = 1001`,
		`SELECT * FROM "Tasks" WHERE TaskStatus = 'WAITING_DEPT_HEAD' AND CreatedBy = 1001 ORDER BY CreatedAt ASC OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY`,
		`SELECT * FROM "Sites" WHERE SiteID IN ('ST1','ST2')`,
	}, rec.sqls)
}

func TestSideBySide_TaskList_ApproverPage2(t *testing.T) {
	h, rec := setup(t, "DEPT_HEAD")

	code, body := get(t, h, "/v1/task?page=2&per_page=5&sort_by=TaskStatus&sort_dir=desc")

	require.Equal(t, http.StatusOK, code, string(body))
	// site-dev-api reported page/per_page from the query but always fetched
	// 20 rows (its Pagination.Limit was never filled). Since Wes, 28 Oct
	// (simple app, API-CHANGES-SIMPLE §2.4) per_page / limit is honoured:
	// offset 5, fetch 5. Same envelope.
	want := oldListEnvelope(t, []oldTaskResp{
		{TaskID: "TS1", SiteID: "ST1", SiteName1: "Toko Kemang", SiteAddress1: "Jl. Kemang Raya 1", SiteAddress3: "Jakarta Selatan",
			TaskStatus: "WAITING_DEPT_HEAD", CreatedBy: 1001, UpdatedBy: 1001, CreatedAt: created, UpdatedAt: updated},
		{TaskID: "TS2", SiteID: "ST2", TaskStatus: "SUBMITED", CreatedBy: 1001, CreatedAt: created, UpdatedAt: created},
	}, 2, 2, 5)
	assertSameJSON(t, want, body)
	assert.Contains(t, rec.sqls, `SELECT * FROM "Tasks" ORDER BY TaskStatus DESC OFFSET 5 ROWS FETCH NEXT 5 ROWS ONLY`)
	assert.Contains(t, rec.sqls, `SELECT count(*) FROM "Tasks"`)
}

func TestSideBySide_TaskHistory(t *testing.T) {
	h, rec := setup(t, "USER")

	code, body := get(t, h, "/v1/task/ST1/history")

	require.Equal(t, http.StatusOK, code, string(body))
	want := oldDataEnvelope(t, []oldTaskHistoryResp{
		{ID: 1, Action: "CREATE", ActorID: 1001, CreatedAt: created},
		{ID: 2, Action: "SUBMIT", ActorID: 1001, Notes: "Site dikirim", CreatedAt: updated},
	})
	assertSameJSON(t, want, body)
	assert.Equal(t, []string{
		`SELECT * FROM "Tasks" WHERE SiteID = 'ST1' ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "TaskHistories" WHERE TaskID = 'TS1' ORDER BY CreatedAt ASC`,
	}, rec.sqls)
}

func TestSideBySide_TaskHistory_UnknownSite(t *testing.T) {
	h, _ := setup(t, "USER")

	code, body := get(t, h, "/v1/task/ST404/history")

	// site-dev-api: gorm.ErrRecordNotFound -> familabs 500 with the raw text
	assert.Equal(t, http.StatusInternalServerError, code)
	assertSameJSON(t, []byte(`{"code":500,"status":"failure","message":"Internal Server Error","errors":{"error":"record not found"}}`), body)
}
