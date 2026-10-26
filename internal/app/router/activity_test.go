package router

// November scope: Activity Log (API-CHANGES §13.4, prototype ActivityResp
// 21 Oct; Dept / Div Head too since Wes 23 Oct) and approver news (prototype ApproverNewsResp 21 Oct). New
// endpoints: shapes and rules from site-dev-mobile src/lib/types.ts and
// mock.ts (activityFor, approverNewsFor).

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

var (
	seenAt  = time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC) // between created and updated
	histCol = []string{"ID", "TaskID", "Action", "ActorID", "Notes", "CreatedAt", "ActorRole", "Round"}
)

func feedSeenRow(userID int64, feed string) sqlfake.Result {
	return sqlfake.Result{Match: `FROM "UserFeedSeen"`, Columns: []string{"UserID", "Feed", "SeenAt"}, Rows: [][]driver.Value{{userID, feed, seenAt}}}
}

func activityResults(role string) []sqlfake.Result {
	return []sqlfake.Result{
		userRow(1001, "andi", "", role),
		userRow(3001, "citra", "", "DEPT_HEAD"),
		feedSeenRow(1001, "ACTIVITY"),
		{Match: `SELECT count(*) FROM "TaskHistories"`, Columns: []string{"count"}, Rows: [][]driver.Value{{int64(3)}}},
		{Match: `SELECT TaskHistories.* FROM "TaskHistories"`, Columns: histCol, Rows: [][]driver.Value{
			{int64(9), "TS1", "REVISION", int64(3001), "x", updated, "DEPT_HEAD", int64(1)},
			{int64(8), "TS2", "APPROVE", int64(3001), "", updated, nil, nil}, // written before ActorRole/Round
			{int64(2), "TS1", "SUBMIT", int64(1001), "", created, "USER", int64(1)},
		}},
		{Match: `FROM "Tasks" WHERE TaskID IN`, Columns: taskColumnsNov, Rows: [][]driver.Value{
			{"TS1", "ST1", "REVISION", int64(1001), int64(3001), created, updated, int64(1)},
			{"TS2", "ST2", "WAITING_DIV_HEAD", int64(1001), int64(3001), created, updated, int64(1)},
		}},
		{Match: `FROM "Sites" WHERE SiteID IN`, Columns: []string{"SiteID", "SiteName1"}, Rows: [][]driver.Value{{"ST1", "Toko Kemang"}, {"ST2", "Toko Dua"}}},
	}
}

func TestActivity_All(t *testing.T) {
	h, rec, _ := stack(t, activityResults("USER")...)

	code, body := call(t, h, http.MethodGet, "/v1/activity?page=1&limit=10", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success","data":[
		{"id":9,"task_id":"TS1","site_id":"ST1","site_name_1":"Toko Kemang","task_status":"REVISION","action":"REVISION","actor_id":3001,"actor_name":"citra","actor_role":"DEPT_HEAD","created_at":"2026-10-03T14:15:00Z","round":1,"needs_action":true,"unread":true},
		{"id":8,"task_id":"TS2","site_id":"ST2","site_name_1":"Toko Dua","task_status":"WAITING_DIV_HEAD","action":"APPROVE","actor_id":3001,"actor_name":"citra","actor_role":"DEPT_HEAD","created_at":"2026-10-03T14:15:00Z","round":1,"needs_action":false,"unread":true},
		{"id":2,"task_id":"TS1","site_id":"ST1","site_name_1":"Toko Kemang","task_status":"REVISION","action":"SUBMIT","actor_id":1001,"actor_name":"andi","actor_role":"USER","created_at":"2026-10-01T09:00:00Z","round":1,"needs_action":true,"unread":false}],"total":3,"page":1,"per_page":10}`), body)
	feed := `FROM "TaskHistories" JOIN Tasks ON Tasks.TaskID = TaskHistories.TaskID WHERE Tasks.CreatedBy = 1001 AND TaskHistories.Action IN ('SUBMIT','RESUBMIT','APPROVE','REVISION','HOLD','RELEASE','REJECT','CANCEL')`
	assert.Equal(t, []string{
		`SELECT * FROM "Users" WHERE UserID = 1001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "UserFeedSeen" WHERE UserID = 1001 AND Feed = 'ACTIVITY' ORDER BY "UserFeedSeen"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT count(*) ` + feed,
		`SELECT TaskHistories.* ` + feed + ` ORDER BY TaskHistories.CreatedAt DESC, TaskHistories.ID DESC OFFSET 0 ROW FETCH NEXT 10 ROWS ONLY`,
	}, rec.sqls[:4])
}

func TestActivity_Filters(t *testing.T) {
	h, rec, _ := stack(t, activityResults("USER")...)

	// action: newest row per request now in REVISION, all rows read, paged here
	code, body := call(t, h, http.MethodGet, "/v1/activity?filter=action", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	var rows []map[string]any
	require.NoError(t, jsonUnmarshalData(body, &rows))
	require.Len(t, rows, 2, "TS1 once (fixture also returns TS2; the status filter is in SQL)")
	assert.Equal(t, float64(9), rows[0]["id"])
	assert.Contains(t, rec.sqls[2], `AND Tasks.TaskStatus = 'REVISION'`)
	assert.NotContains(t, rec.sqls[3], "FETCH NEXT")

	// mine: decisions by others on my requests
	h, rec, _ = stack(t, activityResults("USER")...)
	code, _ = call(t, h, http.MethodGet, "/v1/activity?filter=mine", nil, 1001)
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, rec.sqls[2], `AND TaskHistories.ActorID <> 1001`)
}

func TestActivity_AdminEmpty(t *testing.T) {
	h, rec, _ := stack(t, activityResults("ADMIN")...)

	code, body := call(t, h, http.MethodGet, "/v1/activity", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success","data":[]}`), body) // familabs leaves out total/page when 0
	assert.Len(t, rec.sqls, 1)

	code, body = call(t, h, http.MethodGet, "/v1/activity/unread-count", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success","data":{"count":0}}`), body)
}

// approverActivityResults: Dept Head citra (3001) on TS1 (staff andi sent
// with a note, citra approved, Div Head dewi returned it to the staff) and
// TS2 (citra holds it).
// reader is 3001 (DEPT_HEAD) or 4001 (DIV_HEAD).
func approverActivityResults(reader int64) []sqlfake.Result {
	return []sqlfake.Result{
		userRow(1001, "andi", "", "USER"),
		userRow(3001, "citra", "", "DEPT_HEAD"),
		userRow(4001, "dewi", "", "DIV_HEAD"),
		feedSeenRow(reader, "ACTIVITY"),
		{Match: `SELECT count(*) FROM "TaskHistories"`, Columns: []string{"count"}, Rows: [][]driver.Value{{int64(4)}}},
		{Match: `SELECT TaskHistories.* FROM "TaskHistories"`, Columns: histCol, Rows: [][]driver.Value{
			{int64(12), "TS1", "REVISION", int64(4001), "Site diminta revisi oleh dewi (Div Head) dengan catatan (Foto kurang)", acted, "DIV_HEAD", int64(1)},
			{int64(11), "TS2", "HOLD", int64(3001), "Site ditahan (Hold Status) oleh citra (Dept Head) dengan alasan (Tunggu zonasi)", updated, "DEPT_HEAD", int64(1)},
			{int64(10), "TS1", "APPROVE", int64(3001), "Site di approve oleh citra (Dept Head)", updated, "DEPT_HEAD", int64(1)},
			{int64(2), "TS1", "SUBMIT", int64(1001), "Approval dikirim ke Dept Head, status PENDING (catatan: Dekat stasiun)", created, "USER", int64(1)},
		}},
		{Match: `SELECT * FROM "TaskHistories" WHERE TaskID IN`, Columns: histCol, Rows: [][]driver.Value{
			{int64(10), "TS1", "APPROVE", int64(3001), "", updated, "DEPT_HEAD", int64(1)},
		}},
		{Match: `FROM "Tasks" WHERE TaskID IN`, Columns: taskColumnsNov, Rows: [][]driver.Value{
			{"TS1", "ST1", "REVISION", int64(1001), int64(4001), created, acted, int64(1)},
			{"TS2", "ST2", "HOLD_DEPT_HEAD", int64(1001), int64(3001), created, updated, int64(1)},
		}},
		{Match: `FROM "Sites" WHERE SiteID IN`, Columns: []string{"SiteID", "SiteName1"}, Rows: [][]driver.Value{{"ST1", "Toko Kemang"}, {"ST2", "Toko Dua"}}},
		// TS2 waits for (is held by) citra only
		{Match: `|3001 |PENDING |HOLD |TS`, Columns: []string{"TaskID"}, Rows: [][]driver.Value{{"TS2"}}},
	}
}

// waitingFor is the "now waits for / held by this approver" subquery.
func waitingFor(reader string) string {
	return `(SELECT ApprovalHDR.TaskID FROM "ApprovalDET" JOIN ApprovalHDR ON ApprovalHDR.ID = ApprovalDET.ApprovalHeaderID WHERE ApprovalDET.ApproverID = ` + reader + ` AND ApprovalDET.Status IN ('PENDING','HOLD'))`
}

// approverFeed is the approver Activity Log base: every request sent at
// least once (has an ApprovalHDR round), whoever is in its chain (Wes, 23
// Oct).
const approverFeed = `FROM "TaskHistories" JOIN Tasks ON Tasks.TaskID = TaskHistories.TaskID WHERE TaskHistories.TaskID IN (SELECT ApprovalHDR.TaskID FROM "ApprovalHDR") AND TaskHistories.Action IN ('SUBMIT','RESUBMIT','APPROVE','REVISION','HOLD','RELEASE','REJECT','CANCEL')`

func TestActivity_DeptHead_All(t *testing.T) {
	h, rec, _ := stack(t, approverActivityResults(3001)...)

	code, body := call(t, h, http.MethodGet, "/v1/activity?page=1&limit=10", nil, 3001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success","data":[
		{"id":12,"task_id":"TS1","site_id":"ST1","site_name_1":"Toko Kemang","task_status":"REVISION","action":"REVISION","actor_id":4001,"actor_name":"dewi","actor_role":"DIV_HEAD","created_at":"2026-10-04T10:30:00Z","round":1,"needs_action":false,"unread":true,"notice":"DIV_HEAD_RETURNED"},
		{"id":11,"task_id":"TS2","site_id":"ST2","site_name_1":"Toko Dua","task_status":"HOLD_DEPT_HEAD","action":"HOLD","actor_id":3001,"actor_name":"citra","actor_role":"DEPT_HEAD","created_at":"2026-10-03T14:15:00Z","round":1,"needs_action":true,"unread":false},
		{"id":10,"task_id":"TS1","site_id":"ST1","site_name_1":"Toko Kemang","task_status":"REVISION","action":"APPROVE","actor_id":3001,"actor_name":"citra","actor_role":"DEPT_HEAD","created_at":"2026-10-03T14:15:00Z","round":1,"needs_action":false,"unread":false},
		{"id":2,"task_id":"TS1","site_id":"ST1","site_name_1":"Toko Kemang","task_status":"REVISION","action":"SUBMIT","actor_id":1001,"actor_name":"andi","actor_role":"USER","created_at":"2026-10-01T09:00:00Z","round":1,"needs_action":false,"unread":false}],"total":4,"page":1,"per_page":10}`), body)
	feed := approverFeed
	assert.Equal(t, []string{
		`SELECT * FROM "Users" WHERE UserID = 3001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "UserFeedSeen" WHERE UserID = 3001 AND Feed = 'ACTIVITY' ORDER BY "UserFeedSeen"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT count(*) ` + feed,
		`SELECT TaskHistories.* ` + feed + ` ORDER BY TaskHistories.CreatedAt DESC, TaskHistories.ID DESC OFFSET 0 ROW FETCH NEXT 10 ROWS ONLY`,
	}, rec.sqls[:4])
	assert.Contains(t, rec.sqls, `SELECT ApprovalHDR.TaskID FROM "ApprovalDET" JOIN ApprovalHDR ON ApprovalHDR.ID = ApprovalDET.ApprovalHeaderID WHERE (ApprovalDET.ApproverID = 3001 AND ApprovalDET.Status IN ('PENDING','HOLD')) AND ApprovalHDR.TaskID IN ('TS1','TS2','TS1','TS1')`)
	assert.Equal(t, `SELECT * FROM "TaskHistories" WHERE TaskID IN ('TS1') AND ActorID = 3001 AND Action = 'APPROVE'`, rec.sqls[len(rec.sqls)-1])
}

func TestActivity_DivHead_AllAndFilters(t *testing.T) {
	// Div Head: same feed, the Dept Head's moves too; no "Untuk Anda" notice
	h, rec, _ := stack(t, approverActivityResults(4001)...)
	code, body := call(t, h, http.MethodGet, "/v1/activity", nil, 4001)
	require.Equal(t, http.StatusOK, code, string(body))
	var rows []map[string]any
	require.NoError(t, jsonUnmarshalData(body, &rows))
	require.Len(t, rows, 4)
	assert.Equal(t, "dewi", rows[0]["actor_name"])
	assert.Nil(t, rows[0]["notice"])
	assert.Equal(t, false, rows[0]["unread"], "own move")
	assert.Equal(t, true, rows[1]["unread"], "the Dept Head's hold, after last open")
	assert.Equal(t, false, rows[1]["needs_action"], "held by the Dept Head, not the Div Head")
	assert.Equal(t, `SELECT count(*) `+approverFeed, rec.sqls[2], "all sent requests, no approver filter")
	for _, q := range rec.sqls {
		assert.NotContains(t, q, `ActorID = 4001 AND Action = 'APPROVE'`)
	}

	// action: newest row per request now waiting for / held by this approver
	h, rec, _ = stack(t, approverActivityResults(3001)...)
	code, body = call(t, h, http.MethodGet, "/v1/activity?filter=action", nil, 3001)
	require.Equal(t, http.StatusOK, code, string(body))
	require.NoError(t, jsonUnmarshalData(body, &rows))
	require.Len(t, rows, 2, "TS1 once (fixture also returns other rows; the status filter is in SQL)")
	assert.Equal(t, `SELECT count(*) `+strings.Replace(approverFeed, "WHERE ", "WHERE (", 1)+`) AND TaskHistories.TaskID IN `+waitingFor("3001"), rec.sqls[2])
	assert.NotContains(t, rec.sqls[3], "FETCH NEXT")

	// mine: "Keputusan saya", my own moves
	h, rec, _ = stack(t, approverActivityResults(4001)...)
	code, _ = call(t, h, http.MethodGet, "/v1/activity?filter=mine", nil, 4001)
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, rec.sqls[2], `AND TaskHistories.ActorID = 4001`)
}

func TestActivity_Approver_UnreadCount(t *testing.T) {
	h, rec, _ := stack(t, approverActivityResults(4001)...)

	code, body := call(t, h, http.MethodGet, "/v1/activity/unread-count", nil, 4001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success","data":{"count":4}}`), body)
	assert.Equal(t, `SELECT count(*) FROM "TaskHistories" JOIN Tasks ON Tasks.TaskID = TaskHistories.TaskID WHERE (TaskHistories.TaskID IN (SELECT ApprovalHDR.TaskID FROM "ApprovalHDR") AND TaskHistories.Action IN ('SUBMIT','RESUBMIT','APPROVE','REVISION','HOLD','RELEASE','REJECT','CANCEL')) AND (TaskHistories.ActorID <> 4001 AND TaskHistories.CreatedAt > '2026-10-02 00:00:00')`, rec.sqls[2])
}

func TestActivity_UnreadCountAndSeen(t *testing.T) {
	results := append([]sqlfake.Result{{Match: `SELECT count(*) FROM "TaskHistories"`, Columns: []string{"count"}, Rows: [][]driver.Value{{int64(4)}}}}, activityResults("USER")...)
	h, rec, _ := stack(t, results...)

	code, body := call(t, h, http.MethodGet, "/v1/activity/unread-count", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success","data":{"count":4}}`), body)
	assert.Equal(t, `SELECT count(*) FROM "TaskHistories" JOIN Tasks ON Tasks.TaskID = TaskHistories.TaskID WHERE (Tasks.CreatedBy = 1001 AND TaskHistories.Action IN ('SUBMIT','RESUBMIT','APPROVE','REVISION','HOLD','RELEASE','REJECT','CANCEL')) AND (TaskHistories.ActorID <> 1001 AND TaskHistories.CreatedAt > '2026-10-02 00:00:00')`, rec.sqls[2])

	// first open: no row to update, so one is created
	h, rec, _ = stack(t, userRow(1001, "andi", "", "USER"), sqlfake.Result{Match: `UPDATE "UserFeedSeen"`, Affected: sqlfake.Int64(0)})
	code, body = call(t, h, http.MethodPost, "/v1/activity/seen", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success"}`), body)
	sqls := stableSQL(rec.sqls)
	assert.Equal(t, `UPDATE "UserFeedSeen" SET "SeenAt"='<time>' WHERE UserID = 1001 AND Feed = 'ACTIVITY'`, sqls[1])
	assert.Equal(t, `INSERT INTO "UserFeedSeen" ("UserID","Feed","SeenAt") VALUES (1001,'ACTIVITY','<time>');`, sqls[2])
}

func TestApproverNews(t *testing.T) {
	h, rec, _ := stack(t,
		userRow(3001, "citra", "", "DEPT_HEAD"),
		feedSeenRow(3001, "APPROVER_NEWS"),
		sqlfake.Result{Match: `ApprovalDET.UpdatedAt AS At`, Columns: []string{"TaskID", "SiteID", "At"}, Rows: [][]driver.Value{{"TS2", "ST2", updated}}},
		sqlfake.Result{Match: `ApprovalDET.ActedAt AS At`, Columns: []string{"TaskID", "SiteID", "At"}, Rows: [][]driver.Value{{"TS1", "ST1", acted}}},
	)

	code, body := call(t, h, http.MethodGet, "/v1/task/approver-news", nil, 3001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success","data":[
		{"task_id":"TS1","site_id":"ST1","kind":"DIV_HEAD_RETURNED","created_at":"2026-10-04T10:30:00Z"},
		{"task_id":"TS2","site_id":"ST2","kind":"NEW_TASK","created_at":"2026-10-03T14:15:00Z"}]}`), body)
	joins := `FROM "ApprovalDET" JOIN ApprovalHDR ON ApprovalHDR.ID = ApprovalDET.ApprovalHeaderID JOIN Tasks ON Tasks.TaskID = ApprovalHDR.TaskID JOIN Sites ON Sites.SiteID = Tasks.SiteID`
	assert.Equal(t, `SELECT ApprovalHDR.TaskID, Tasks.SiteID, ApprovalDET.UpdatedAt AS At `+joins+` WHERE ApprovalDET.ApproverID = 3001 AND ApprovalDET.Status = 'PENDING' AND ApprovalDET.UpdatedAt > '2026-10-02 00:00:00'`, rec.sqls[2])
	assert.Equal(t, `SELECT ApprovalHDR.TaskID, Tasks.SiteID, ApprovalDET.ActedAt AS At `+joins+` WHERE (ApprovalDET.Status = 'REVISION' AND ApprovalDET.ApproverRoleID = 'DIV_HEAD' AND ApprovalDET.ActedAt > '2026-10-02 00:00:00') AND EXISTS (SELECT 1 FROM ApprovalDET AS mine WHERE mine.ApprovalHeaderID = ApprovalDET.ApprovalHeaderID AND mine.ApproverID = 3001 AND mine.Status = 'APPROVED' AND mine.Level < ApprovalDET.Level)`, rec.sqls[3])

	// staff: nothing
	h, _, _ = stack(t, userRow(1001, "andi", "", "USER"))
	code, body = call(t, h, http.MethodGet, "/v1/task/approver-news", nil, 1001)
	require.Equal(t, http.StatusOK, code)
	assertSameJSON(t, []byte(`{"code":200,"status":"success","data":[]}`), body)

	h, rec, _ = stack(t, userRow(3001, "citra", "", "DEPT_HEAD"))
	code, _ = call(t, h, http.MethodPost, "/v1/task/approver-news/seen", nil, 3001)
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, stableSQL(rec.sqls)[1], `WHERE UserID = 3001 AND Feed = 'APPROVER_NEWS'`)
}

// A head outside the requests' approval chain (second Dept Head 3002: no
// ApprovalDET row anywhere) still sees every sent request (Wes, 23 Oct);
// "Perlu tindakan" stays his own (TS2 is held by citra, not by him), and
// no "Untuk Anda" since he approved nothing.
func TestActivity_HeadOutsideChain_SeesAllSent(t *testing.T) {
	results := append([]sqlfake.Result{userRow(3002, "eko", "", "DEPT_HEAD"), feedSeenRow(3002, "ACTIVITY"),
		{Match: `SELECT * FROM "TaskHistories" WHERE TaskID IN`, Columns: histCol}}, approverActivityResults(3002)...)
	h, rec, _ := stack(t, results...)

	code, body := call(t, h, http.MethodGet, "/v1/activity", nil, 3002)

	require.Equal(t, http.StatusOK, code, string(body))
	var rows []map[string]any
	require.NoError(t, jsonUnmarshalData(body, &rows))
	require.Len(t, rows, 4)
	assert.Equal(t, `SELECT count(*) `+approverFeed, rec.sqls[2], "no approver filter on the feed")
	assert.Equal(t, `SELECT TaskHistories.* `+approverFeed+` ORDER BY TaskHistories.CreatedAt DESC, TaskHistories.ID DESC OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY`, rec.sqls[3])
	assert.Nil(t, rows[0]["notice"], "eko approved nothing")
	for _, r := range rows {
		assert.NotContains(t, r, "notes", "no note/reason on Activity Log rows (Wes, 23 Oct)")
	}
	assert.Equal(t, true, rows[1]["unread"], "citra's hold is someone else's move")
	assert.Equal(t, false, rows[1]["needs_action"], "TS2 is held by citra, not by eko")
}
