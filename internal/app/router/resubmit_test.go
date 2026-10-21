package router

// November scope, API-CHANGES §13.3: resubmit after REVISION (history
// RESUBMIT, Tasks.SubmitCount + 1), history actor_role + round,
// submit_count and rejected_by_role on the lists and the detail. New
// behaviour (no site-dev-api answer to compare with): shapes from the
// prototype (site-dev-mobile src/lib/types.ts TaskResp, TaskHistoryResp).

import (
	"database/sql/driver"
	"net/http"
	"testing"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var taskColumnsNov = append(append([]string(nil), taskColumns...), "SubmitCount")

func TestResubmit_CountsUpAndWritesResubmit(t *testing.T) {
	m := withFlow(t)
	m.mail.On("SendApprovalPending", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	revisionTS1 := []driver.Value{"TS1", "ST1", "REVISION", int64(1001), int64(3002), created, updated, int64(2)}
	results := append([]sqlfake.Result{
		{Match: `FROM "Tasks" WHERE TaskID = @p1`, Columns: taskColumnsNov, Rows: [][]driver.Value{revisionTS1}},
		{Match: `FROM "Tasks" WHERE SiteID = @p1`, Columns: taskColumnsNov, Rows: [][]driver.Value{revisionTS1}},
		chainResult(),
		roleRow("DEPT_HEAD", "Dept Head", 1),
		{Match: `INSERT INTO "ApprovalDET"`, Columns: []string{"ID"}, Rows: [][]driver.Value{{int64(31)}, {int64(32)}}},
	}, saveResults("REVISION")...)
	h, rec, _ := stack(t, results...)

	code, body := call(t, h, http.MethodPost, "/v1/task/SUBMITED", map[string]any{"site": map[string]any{"site_id": "ST1", "site_name_1": "Toko Kemang"}}, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	sqls := stableSQL(rec.sqls)
	assert.Contains(t, sqls, `UPDATE "Tasks" SET "SubmitCount"=3,"TaskStatus"='WAITING_DEPT_HEAD',"UpdatedAt"='<time>',"UpdatedBy"=1001 WHERE TaskID = 'TS1'`)
	assert.Contains(t, sqls, `INSERT INTO "TaskHistories" ("TaskID","Action","ActorID","Notes","CreatedAt","ActorRole","Round") OUTPUT INSERTED."ID" VALUES ('TS1','RESUBMIT',1001,'Approval dikirim ke Dept Head, status PENDING (catatan: Site diajukan oleh andi untuk proses approval)','<time>','USER',3);`)
}

func TestHistory_ActorRoleAndRound(t *testing.T) {
	h, _, _ := stack(t,
		userRow(1001, "andi", "", "USER"),
		sqlfake.Result{Match: `FROM "Tasks" WHERE SiteID = @p1`, Columns: taskColumns, Rows: [][]driver.Value{{"TS1", "ST1", "REVISION", int64(1001), int64(1001), created, updated}}},
		sqlfake.Result{Match: `FROM "TaskHistories"`, Columns: []string{"ID", "TaskID", "Action", "ActorID", "Notes", "CreatedAt", "ActorRole", "Round"},
			Rows: [][]driver.Value{
				{int64(1), "TS1", "SUBMIT", int64(1001), "lama", created, nil, nil}, // written before the change
				{int64(2), "TS1", "REVISION", int64(3002), "x", updated, "DIV_HEAD", int64(1)},
			}},
	)

	code, body := call(t, h, http.MethodGet, "/v1/task/ST1/history", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success","data":[
		{"id":1,"action":"SUBMIT","actor_id":1001,"notes":"lama","created_at":"2026-10-01T09:00:00Z"},
		{"id":2,"action":"REVISION","actor_id":3002,"notes":"x","created_at":"2026-10-03T14:15:00Z","actor_role":"DIV_HEAD","round":1}]}`), body)
}

func TestTaskList_SubmitCountAndRejectedByRole(t *testing.T) {
	h, rec, _ := stack(t,
		userRow(1001, "andi", "", "USER"),
		sqlfake.Result{Match: `SELECT count(*) FROM "Tasks"`, Columns: []string{"count"}, Rows: [][]driver.Value{{int64(2)}}},
		sqlfake.Result{Match: `FROM "Tasks"`, Columns: taskColumnsNov, Rows: [][]driver.Value{
			{"TS1", "ST1", "REVISION", int64(1001), int64(3002), created, updated, int64(2)},
			{"TS2", "ST2", "WAITING_DEPT_HEAD", int64(1001), int64(1001), created, updated, int64(1)},
		}},
		sqlfake.Result{Match: `FROM "Sites"`, Columns: []string{"SiteID", "SiteName1"}, Rows: [][]driver.Value{{"ST1", "Toko Kemang"}, {"ST2", "Toko Dua"}}},
		sqlfake.Result{Match: `JOIN ApprovalHDR`, Columns: []string{"TaskID", "ApproverRoleID", "ActedAt"}, Rows: [][]driver.Value{
			{"TS1", "DEPT_HEAD", created}, {"TS1", "DIV_HEAD", updated}, // round 1 by Dept Head, round 2 by Div Head
		}},
	)

	code, body := call(t, h, http.MethodGet, "/v1/task", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	var rows []map[string]any
	require.NoError(t, jsonUnmarshalData(body, &rows))
	require.Len(t, rows, 2)
	assert.Equal(t, float64(2), rows[0]["submit_count"])
	assert.Equal(t, "DIV_HEAD", rows[0]["rejected_by_role"])
	assert.Equal(t, float64(1), rows[1]["submit_count"])
	assert.NotContains(t, rows[1], "rejected_by_role")
	assert.Equal(t, `SELECT ApprovalHDR.TaskID, ApprovalDET.ApproverRoleID, ApprovalDET.ActedAt FROM "ApprovalDET" JOIN ApprovalHDR ON ApprovalHDR.ID = ApprovalDET.ApprovalHeaderID WHERE ApprovalHDR.TaskID IN ('TS1') AND ApprovalDET.Status IN ('REVISION','REJECTED') ORDER BY ApprovalDET.ActedAt ASC`, rec.sqls[len(rec.sqls)-1])
}

func TestTaskList_NoReturnedTaskNoExtraQuery(t *testing.T) {
	h, rec := setup(t, "USER")

	code, _ := get(t, h, "/v1/task")

	require.Equal(t, http.StatusOK, code)
	for _, s := range rec.sqls {
		assert.NotContains(t, s, "JOIN ApprovalHDR")
	}
}

func TestTaskDetail_SubmitCountAndRejectedByRole(t *testing.T) {
	results := append([]sqlfake.Result{
		{Match: `FROM "Tasks" WHERE TaskID = @p1`, Columns: taskColumnsNov, Rows: [][]driver.Value{{"TS1", "ST1", "REVISION", int64(1001), int64(3001), created, updated, int64(2)}}},
		{Match: `FROM "ApprovalDET"`, Columns: detailCols, Rows: [][]driver.Value{
			{int64(31), int64(7), int64(1), "DEPT_HEAD", int64(3001), "REVISION", "foto kurang", acted, "C1", updated, acted},
			{int64(32), int64(7), int64(2), "DIV_HEAD", int64(3002), "WAITING", "", nil, "C2", updated, updated},
		}},
	}, detailResults("USER")...)
	h, _, _ := stack(t, results...)

	code, body := call(t, h, http.MethodGet, "/v1/task/TS1", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	var detail struct {
		Task map[string]any `json:"task"`
	}
	require.NoError(t, jsonUnmarshalData(body, &detail))
	assert.Equal(t, float64(2), detail.Task["submit_count"])
	assert.Equal(t, "DEPT_HEAD", detail.Task["rejected_by_role"])
}
