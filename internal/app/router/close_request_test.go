package router

// Wes, 26 Oct: one "Tolak" pop-up with two outcomes (REVISION "Tolak, bisa
// direvisi" / REJECT "Tolak final"), both with optional notes; the staff
// "Hapus draft" (DELETE /v1/site/:id) and "Batalkan pengajuan" (POST
// /v1/task/:id/cancel). New behaviour: no site-dev-api answer to compare.

import (
	"database/sql/driver"
	"net/http"
	"strings"
	"testing"

	"site-dev-backend/internal/app/client"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const historyInsert = `INSERT INTO "TaskHistories" ("TaskID","Action","ActorID","Notes","CreatedAt","ActorRole","Round") OUTPUT INSERTED."ID" VALUES `

func TestReject_WithoutNotes(t *testing.T) {
	withFlow(t)
	h, rec, _ := stack(t, approveResults(1, 2)...)

	code, body := call(t, h, http.MethodPost, "/v1/task/TS1/approve", map[string]any{"decision": "REJECT"}, 3001)
	require.Equal(t, http.StatusOK, code, string(body))
	assert.Equal(t, []string{
		`UPDATE "ApprovalDET" SET "ActedAt"='<time>',"Notes"='',"Status"='REJECTED',"UpdatedAt"='<time>' WHERE ID = 31 AND Status = 'PENDING'`,
		`SELECT * FROM "Roles" WHERE RoleID = 'DEPT_HEAD' ORDER BY "Roles"."RoleID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		historyInsert + `('TS1','REJECT',3001,'Site di reject oleh citra (Dept Head)','<time>','DEPT_HEAD',1);`,
		`UPDATE "ApprovalHDR" SET "OverallStatus"='REJECTED',"UpdatedAt"='<time>' WHERE ID = 7`,
		`UPDATE "Tasks" SET "TaskStatus"='REJECTED',"UpdatedAt"='<time>',"UpdatedBy"=3001 WHERE TaskID = 'TS1'`,
	}, stableSQL(rec.sqls)[4:])

	rec.sqls = nil
	code, body = call(t, h, http.MethodPost, "/v1/task/TS1/approve", map[string]any{"decision": "REVISION", "notes": ""}, 3001)
	require.Equal(t, http.StatusOK, code, string(body))
	assert.Contains(t, stableSQL(rec.sqls), historyInsert+`('TS1','REVISION',3001,'Site diminta revisi oleh citra (Dept Head)','<time>','DEPT_HEAD',1);`)
	assert.Contains(t, stableSQL(rec.sqls), `UPDATE "Tasks" SET "TaskStatus"='REVISION',"UpdatedAt"='<time>',"UpdatedBy"=3001 WHERE TaskID = 'TS1'`)
}

func TestTaskList_StatusRejectedGroup(t *testing.T) {
	h, rec := setup(t, "USER")

	code, body := get(t, h, "/v1/task?status=REJECTED_ANY")

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Equal(t, `SELECT count(*) FROM "Tasks" WHERE TaskStatus IN ('REVISION','REJECTED') AND CreatedBy = 1001`, rec.sqls[1])
	assert.Equal(t, `SELECT * FROM "Tasks" WHERE TaskStatus IN ('REVISION','REJECTED') AND CreatedBy = 1001 ORDER BY CreatedAt ASC OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY`, rec.sqls[2])
}

func TestTaskList_ActiveViaStatusNot(t *testing.T) {
	h, rec := setup(t, "USER")

	code, body := get(t, h, "/v1/task?status_not=REJECTED,CANCELLED")

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Equal(t, `SELECT count(*) FROM "Tasks" WHERE TaskStatus NOT IN ('REJECTED','CANCELLED') AND CreatedBy = 1001`, rec.sqls[1])
}

func taskRow(status string, createdBy int64) []driver.Value {
	return []driver.Value{"TS1", "ST1", status, createdBy, int64(3001), created, updated, int64(2)}
}

func cancelResults(status string, createdBy int64) []sqlfake.Result {
	return []sqlfake.Result{
		userRow(1001, "andi", "andi@example.com", "USER"),
		userRow(1002, "bayu", "bayu@example.com", "USER"),
		roleRow("USER", "Staf", 0),
		{Match: `FROM "Tasks" WHERE TaskID = @p1`, Columns: taskColumnsNov, Rows: [][]driver.Value{taskRow(status, createdBy)}},
		{Match: `FROM "Tasks" WHERE SiteID = @p1`, Columns: taskColumnsNov, Rows: [][]driver.Value{taskRow(status, createdBy)}},
	}
}

func TestCancel(t *testing.T) {
	h, rec, fdb := stack(t, cancelResults("REVISION", 1001)...)

	code, body := call(t, h, http.MethodPost, "/v1/task/TS1/cancel", map[string]any{"notes": "  lokasi sudah diambil pesaing "}, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success"}`), body)
	assert.Equal(t, []string{
		`SELECT * FROM "Users" WHERE UserID = 1001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Tasks" WHERE TaskID = 'TS1' ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`UPDATE "Tasks" SET "TaskStatus"='CANCELLED',"UpdatedAt"='<time>',"UpdatedBy"=1001 WHERE TaskID = 'TS1' AND TaskStatus = 'REVISION'`,
		`SELECT * FROM "Roles" WHERE RoleID = 'USER' ORDER BY "Roles"."RoleID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		historyInsert + `('TS1','CANCEL',1001,'Pengajuan dibatalkan oleh andi (Staf) dengan catatan (lokasi sudah diambil pesaing)','<time>','USER',2);`,
	}, stableSQL(rec.sqls))
	assert.Equal(t, "COMMIT", fdb.Queries()[len(fdb.Queries())-1])

	// no body, by site_id
	rec.sqls = nil
	code, body = call(t, h, http.MethodPost, "/v1/task/ST1/cancel", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assert.Contains(t, stableSQL(rec.sqls), historyInsert+`('TS1','CANCEL',1001,'Pengajuan dibatalkan oleh andi (Staf)','<time>','USER',2);`)
}

func TestCancel_Refusals(t *testing.T) {
	cases := []struct {
		name   string
		status string
		owner  int64
		user   int64
		path   string
		body   any
		code   int
		msg    string
	}{
		{"not the creator", "REVISION", 1001, 1002, "/v1/task/TS1/cancel", nil, http.StatusForbidden, "Hanya pembuat pengajuan yang bisa membatalkannya."},
		{"waiting", "WAITING_DEPT_HEAD", 1001, 1001, "/v1/task/TS1/cancel", nil, http.StatusConflict, "Hanya pengajuan yang Ditolak (bisa direvisi) yang bisa dibatalkan."},
		{"hold", "HOLD_DEPT_HEAD", 1001, 1001, "/v1/task/TS1/cancel", nil, http.StatusConflict, "Hanya pengajuan yang Ditolak (bisa direvisi) yang bisa dibatalkan."},
		{"final rejected", "REJECTED", 1001, 1001, "/v1/task/TS1/cancel", nil, http.StatusConflict, "Hanya pengajuan yang Ditolak (bisa direvisi) yang bisa dibatalkan."},
		{"already cancelled", "CANCELLED", 1001, 1001, "/v1/task/TS1/cancel", nil, http.StatusConflict, "Hanya pengajuan yang Ditolak (bisa direvisi) yang bisa dibatalkan."},
		{"unknown id", "REVISION", 1001, 1001, "/v1/task/XX1/cancel", nil, http.StatusNotFound, "Pengajuan tidak ditemukan."},
		{"notes too long", "REVISION", 1001, 1001, "/v1/task/TS1/cancel", map[string]any{"notes": strings.Repeat("é", 501)}, http.StatusUnprocessableEntity, ""},
	}
	for _, tc := range cases {
		h, rec, _ := stack(t, cancelResults(tc.status, tc.owner)...)
		code, body := call(t, h, http.MethodPost, tc.path, tc.body, tc.user)
		assert.Equal(t, tc.code, code, tc.name)
		if tc.msg != "" {
			assert.Contains(t, string(body), tc.msg, tc.name)
		}
		for _, q := range rec.sqls {
			assert.NotContains(t, q, "UPDATE", tc.name)
			assert.NotContains(t, q, "INSERT", tc.name)
		}
	}

	// changed meanwhile (resubmitted): 409, rolled back
	conflict := sqlfake.Result{Match: `UPDATE "Tasks"`, Affected: sqlfake.Int64(0)}
	h, _, fdb := stack(t, append([]sqlfake.Result{conflict}, cancelResults("REVISION", 1001)...)...)
	code, _ := call(t, h, http.MethodPost, "/v1/task/TS1/cancel", nil, 1001)
	assert.Equal(t, http.StatusConflict, code)
	assert.Contains(t, fdb.Queries(), "ROLLBACK")
}

var attachmentCols = []string{"SiteAttachmentID", "SiteID", "FileName", "Path", "CreatedAt"}

func deleteResults(status string, owner int64, header bool, actions ...string) []sqlfake.Result {
	site := siteST1Row()
	site[21] = owner // CreatedBy
	res := []sqlfake.Result{
		userRow(1001, "andi", "andi@example.com", "USER"),
		userRow(1002, "bayu", "bayu@example.com", "USER"),
		{Match: `FROM "Tasks" WHERE TaskID = @p1`, Columns: taskColumnsNov, Rows: [][]driver.Value{taskRow(status, owner)}},
		{Match: `FROM "Tasks" WHERE SiteID = @p1`, Columns: taskColumnsNov, Rows: [][]driver.Value{taskRow(status, owner)}},
		{Match: `FROM "Sites" WHERE SiteID = @p1`, Columns: siteColumns, Rows: [][]driver.Value{site}},
		{Match: `FROM "SiteAttachments" WHERE SiteID`, Columns: attachmentCols, Rows: [][]driver.Value{
			{int64(41), "ST1", "depan.jpg", "test-bucket/site-dev/ST1/depan.jpg", created},
			{int64(42), "ST1", "rusak.jpg", "no-slash", created},
		}},
	}
	if header {
		res = append(res, sqlfake.Result{Match: `FROM "ApprovalHDR" WHERE TaskID`, Columns: []string{"ID", "TaskID", "OverallStatus"}, Rows: [][]driver.Value{{int64(7), "TS1", "PENDING"}}})
	} else {
		res = append(res, noRows(`FROM "ApprovalHDR" WHERE TaskID`))
	}
	rows := [][]driver.Value{{int64(1), "TS1", "CREATE", owner, "Site baru dibuat oleh andi", created}}
	for i, a := range actions {
		rows = append(rows, []driver.Value{int64(2 + i), "TS1", a, owner, "", updated})
	}
	return append(res, sqlfake.Result{Match: `FROM "TaskHistories" WHERE TaskID`, Columns: []string{"ID", "TaskID", "Action", "ActorID", "Notes", "CreatedAt"}, Rows: rows})
}

func TestDeleteDraft(t *testing.T) {
	storage := mocks.NewIStorageClient(t)
	withClients(t, func(c *client.Clients) { c.Storage = storage })
	storage.On("Delete", mock.Anything, "test-bucket", "site-dev/ST1/depan.jpg").Return(nil).Once()
	h, rec, fdb := stack(t, deleteResults("DRAFT", 1001, false, "UPDATE")...)

	code, body := call(t, h, http.MethodDelete, "/v1/site/ST1", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success"}`), body)
	sqls := stableSQL(rec.sqls)
	assert.Equal(t, []string{
		`SELECT * FROM "Users" WHERE UserID = 1001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Tasks" WHERE SiteID = 'ST1' ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Sites" WHERE SiteID = 'ST1' ORDER BY "Sites"."SiteID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "ApprovalHDR" WHERE TaskID = 'TS1' ORDER BY ID DESC OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "TaskHistories" WHERE TaskID = 'TS1' ORDER BY CreatedAt ASC`,
		`SELECT * FROM "SiteAttachments" WHERE SiteID = 'ST1' ORDER BY SiteAttachmentID ASC`,
		`UPDATE "Tasks" SET "UpdatedBy"=1001,"UpdatedAt"='<time>' WHERE TaskID = 'TS1' AND TaskStatus = 'DRAFT'`,
		`DELETE FROM SiteAIMetrics WHERE SiteAIID IN (SELECT SiteAIID FROM SiteAIInsights WHERE SiteID = 'ST1')`,
		`DELETE FROM SiteAIRisks WHERE SiteAIID IN (SELECT SiteAIID FROM SiteAIInsights WHERE SiteID = 'ST1')`,
		`DELETE FROM SiteAIOpportunities WHERE SiteAIID IN (SELECT SiteAIID FROM SiteAIInsights WHERE SiteID = 'ST1')`,
		`DELETE FROM SiteAIDataAssumptions WHERE SiteAIID IN (SELECT SiteAIID FROM SiteAIInsights WHERE SiteID = 'ST1')`,
		`DELETE FROM SiteAIContextWarnings WHERE SiteAIID IN (SELECT SiteAIID FROM SiteAIInsights WHERE SiteID = 'ST1')`,
		`DELETE FROM SiteAIInsights WHERE SiteID = 'ST1'`,
		`DELETE FROM SiteAttachments WHERE SiteID = 'ST1'`,
		`DELETE FROM SiteCompetitors WHERE SiteID = 'ST1'`,
		`DELETE FROM SiteCustomNotes WHERE SiteID = 'ST1'`,
		`DELETE FROM SiteContexts WHERE SiteID = 'ST1'`,
		`DELETE FROM SiteAds WHERE SiteID = 'ST1'`,
		`DELETE FROM TaskHistories WHERE TaskID = 'TS1'`,
		`DELETE FROM Tasks WHERE TaskID = 'TS1'`,
		`DELETE FROM Sites WHERE SiteID = 'ST1'`,
	}, sqls)
	assert.Contains(t, fdb.Queries(), "COMMIT")
	// the photo with a bad path is logged and skipped; the delete still succeeds
}

func TestDeleteDraft_Refusals(t *testing.T) {
	cases := []struct {
		name    string
		status  string
		owner   int64
		user    int64
		header  bool
		actions []string
		path    string
		code    int
		msg     string
	}{
		{"not the creator", "DRAFT", 1001, 1002, false, nil, "/v1/site/ST1", http.StatusForbidden, "Hanya pembuat draft yang bisa menghapusnya."},
		{"sent and waiting", "WAITING_DEPT_HEAD", 1001, 1001, true, []string{"SUBMIT"}, "/v1/site/ST1", http.StatusConflict, "Hanya draft yang belum pernah dikirim yang bisa dihapus."},
		{"returned (REVISION)", "REVISION", 1001, 1001, true, []string{"SUBMIT", "REVISION"}, "/v1/site/TS1", http.StatusConflict, "Hanya draft yang belum pernah dikirim yang bisa dihapus."},
		{"DRAFT with an approval round", "DRAFT", 1001, 1001, true, nil, "/v1/site/ST1", http.StatusConflict, "Hanya draft yang belum pernah dikirim yang bisa dihapus."},
		{"DRAFT with a SUBMIT row", "DRAFT", 1001, 1001, false, []string{"SUBMIT"}, "/v1/site/ST1", http.StatusConflict, "Hanya draft yang belum pernah dikirim yang bisa dihapus."},
		{"unknown id", "DRAFT", 1001, 1001, false, nil, "/v1/site/abc", http.StatusNotFound, "Pengajuan tidak ditemukan."},
	}
	for _, tc := range cases {
		h, rec, _ := stack(t, deleteResults(tc.status, tc.owner, tc.header, tc.actions...)...)
		code, body := call(t, h, http.MethodDelete, tc.path, nil, tc.user)
		assert.Equal(t, tc.code, code, tc.name)
		assert.Contains(t, string(body), tc.msg, tc.name)
		for _, q := range rec.sqls {
			assert.NotContains(t, q, "DELETE", tc.name)
			assert.NotContains(t, q, "UPDATE", tc.name)
		}
	}

	// sent meanwhile: 409, rolled back, nothing deleted
	conflict := sqlfake.Result{Match: `UPDATE "Tasks"`, Affected: sqlfake.Int64(0)}
	h, rec, fdb := stack(t, append([]sqlfake.Result{conflict}, deleteResults("DRAFT", 1001, false)...)...)
	code, _ := call(t, h, http.MethodDelete, "/v1/site/ST1", nil, 1001)
	assert.Equal(t, http.StatusConflict, code)
	assert.Contains(t, fdb.Queries(), "ROLLBACK")
	for _, q := range rec.sqls {
		assert.NotContains(t, q, "DELETE")
	}
}
