package router

// November scope, API-CHANGES §13.1/§13.3: HOLD ("Tahan") and RELEASE
// ("Lanjutkan") on POST /v1/task/:id/approve, and status=HOLD on the list.
// New behaviour: there is no site-dev-api answer to compare with; the
// rules are the prototype's (site-dev-mobile mock.ts actOnApproval).

import (
	"database/sql/driver"
	"net/http"
	"testing"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func heldResults() []sqlfake.Result {
	return append([]sqlfake.Result{
		noRows(`|7 |3001 |PENDING`),
		{Match: `|7 |3001 |HOLD`, Columns: detailCols, Rows: [][]driver.Value{{int64(31), int64(7), int64(1), "DEPT_HEAD", int64(3001), "HOLD", "tunggu zonasi", acted, "CODE31", updated, acted}}},
	}, approveResults(1, 2)...)
}

func TestHold(t *testing.T) {
	withFlow(t) // no e-mail is expected (the mail mock fails on any call)
	h, rec, fdb := stack(t, approveResults(1, 2)...)

	code, body := call(t, h, http.MethodPost, "/v1/task/TS1/approve", map[string]any{"decision": "HOLD", "notes": "tunggu zonasi"}, 3001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success"}`), body)
	assert.Equal(t, []string{
		`UPDATE "ApprovalDET" SET "ActedAt"='<time>',"Notes"='tunggu zonasi',"Status"='HOLD',"UpdatedAt"='<time>' WHERE ID = 31 AND Status = 'PENDING'`,
		`SELECT * FROM "Roles" WHERE RoleID = 'DEPT_HEAD' ORDER BY "Roles"."RoleID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`INSERT INTO "TaskHistories" ("TaskID","Action","ActorID","Notes","CreatedAt") OUTPUT INSERTED."ID" VALUES ('TS1','HOLD',3001,'Site ditahan (Hold Status) oleh citra (Dept Head) dengan alasan (tunggu zonasi)','<time>');`,
		`UPDATE "Tasks" SET "TaskStatus"='HOLD_DEPT_HEAD',"UpdatedAt"='<time>',"UpdatedBy"=3001 WHERE TaskID = 'TS1'`,
	}, stableSQL(rec.sqls)[4:])
	assert.Equal(t, "COMMIT", fdb.Queries()[len(fdb.Queries())-1], "the round stays PENDING, nothing after the commit")

	code, body = call(t, h, http.MethodPost, "/v1/task/TS1/approve", map[string]any{"decision": "HOLD"}, 3001)
	assert.Equal(t, http.StatusUnprocessableEntity, code)
	assert.Contains(t, string(body), `"notes":"Alasan Hold Status wajib diisi."`)
}

func TestRelease(t *testing.T) {
	withFlow(t)
	h, rec, _ := stack(t, heldResults()...)

	code, body := call(t, h, http.MethodPost, "/v1/task/TS1/approve", map[string]any{"decision": "RELEASE"}, 3001)

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Equal(t, []string{
		`SELECT * FROM "ApprovalDET" WHERE ApprovalHeaderID = 7 AND ApproverID = 3001 AND Status = 'HOLD' ORDER BY "ApprovalDET"."ID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Users" WHERE UserID = 3001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`UPDATE "ApprovalDET" SET "ActedAt"=NULL,"Notes"='',"Status"='PENDING',"UpdatedAt"='<time>' WHERE ID = 31 AND Status = 'HOLD'`,
		`SELECT * FROM "Roles" WHERE RoleID = 'DEPT_HEAD' ORDER BY "Roles"."RoleID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`INSERT INTO "TaskHistories" ("TaskID","Action","ActorID","Notes","CreatedAt") OUTPUT INSERTED."ID" VALUES ('TS1','RELEASE',3001,'Hold Status dilepas oleh citra (Dept Head)','<time>');`,
		`UPDATE "Tasks" SET "TaskStatus"='WAITING_DEPT_HEAD',"UpdatedAt"='<time>',"UpdatedBy"=3001 WHERE TaskID = 'TS1'`,
	}, stableSQL(rec.sqls)[2:])
}

func TestHold_Refusals(t *testing.T) {
	withFlow(t)
	h, _, _ := stack(t, heldResults()...)

	for _, decision := range []string{"APPROVE", "REVISION", "HOLD"} {
		code, body := call(t, h, http.MethodPost, "/v1/task/TS1/approve", map[string]any{"decision": decision, "notes": "x"}, 3001)
		assert.Equal(t, http.StatusBadRequest, code, decision)
		assertSameJSON(t, []byte(`{"code":400,"status":"failure","message":"Bad Request","errors":{"error":"Pengajuan ini sedang Anda tahan. Tekan Lanjutkan dulu."}}`), body)
	}

	h, _, _ = stack(t, append([]sqlfake.Result{noRows(`|7 |3001 |HOLD`)}, approveResults(1, 2)...)...)
	code, body := call(t, h, http.MethodPost, "/v1/task/TS1/approve", map[string]any{"decision": "RELEASE"}, 3001)
	assert.Equal(t, http.StatusBadRequest, code)
	assertSameJSON(t, []byte(`{"code":400,"status":"failure","message":"Bad Request","errors":{"error":"Pengajuan ini tidak sedang dalam Hold Status."}}`), body)
}

func TestTaskList_StatusHoldGroup(t *testing.T) {
	h, rec := setup(t, "USER")

	code, body := get(t, h, "/v1/task?status=HOLD")

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Equal(t, `SELECT count(*) FROM "Tasks" WHERE TaskStatus IN ('HOLD_DEPT_HEAD','HOLD_DIV_HEAD') AND CreatedBy = 1001`, rec.sqls[1])
	assert.Equal(t, `SELECT * FROM "Tasks" WHERE TaskStatus IN ('HOLD_DEPT_HEAD','HOLD_DIV_HEAD') AND CreatedBy = 1001 ORDER BY CreatedAt ASC OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY`, rec.sqls[2])
}
