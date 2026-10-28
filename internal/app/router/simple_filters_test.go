package router

// Simple app (Wes, 28 Oct, API-CHANGES-SIMPLE.md): list filters the simple
// version needs on GET /v1/task.

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// status=WAITING = "Menunggu": both waiting levels in one paged list (§2.2).
func TestTaskList_StatusWaitingGroup(t *testing.T) {
	h, rec := setup(t, "USER")

	code, body := get(t, h, "/v1/task?status=WAITING")

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Equal(t, `SELECT count(*) FROM "Tasks" WHERE TaskStatus IN ('SUBMITED','WAITING_DEPT_HEAD','WAITING_DIV_HEAD') AND CreatedBy = 1001`, rec.sqls[1])
	assert.Equal(t, `SELECT * FROM "Tasks" WHERE TaskStatus IN ('SUBMITED','WAITING_DEPT_HEAD','WAITING_DIV_HEAD') AND CreatedBy = 1001 ORDER BY CreatedAt ASC OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY`, rec.sqls[2])
}

// Heads see the waiting requests of everyone (same scope as every list).
func TestTaskList_StatusWaitingGroup_Head(t *testing.T) {
	h, rec := setup(t, "DEPT_HEAD")

	code, body := get(t, h, "/v1/task?status=WAITING")

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Contains(t, rec.sqls, `SELECT count(*) FROM "Tasks" WHERE TaskStatus IN ('SUBMITED','WAITING_DEPT_HEAD','WAITING_DIV_HEAD')`)
	for _, sql := range rec.sqls {
		assert.NotContains(t, sql, "CreatedBy =", "heads are not limited to their own requests")
	}
}

// limit / per_page on GET /v1/task (§2.4): default 20, max 100; without
// either the answer is the old one (20 rows).
func TestTaskList_Limit(t *testing.T) {
	for _, c := range []struct{ query, fetch string }{
		{"/v1/task?status=WAITING&limit=8&page=2", "OFFSET 8 ROWS FETCH NEXT 8 ROWS ONLY"},
		{"/v1/task?per_page=8", "OFFSET 0 ROW FETCH NEXT 8 ROWS ONLY"},
		{"/v1/task?limit=500", "OFFSET 0 ROW FETCH NEXT 100 ROWS ONLY"},
		{"/v1/task?limit=0", "OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY"},
		{"/v1/task", "OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY"},
	} {
		h, rec := setup(t, "USER")
		code, body := get(t, h, c.query)
		require.Equal(t, http.StatusOK, code, string(body))
		require.Len(t, rec.sqls, 4, c.query)
		assert.True(t, strings.HasSuffix(rec.sqls[2], c.fetch), "%s: %s", c.query, rec.sqls[2])
	}
}

func TestTaskList_LimitEnvelope(t *testing.T) {
	h, _ := setup(t, "DEPT_HEAD")
	code, body := get(t, h, "/v1/task?limit=8")
	require.Equal(t, http.StatusOK, code, string(body))
	assert.Contains(t, string(body), `"page":1,"per_page":8`)
}

// the heads' lists take the same page size (§2.4)
func TestApprovalLists_Limit(t *testing.T) {
	for _, path := range []string{"/v1/task/pending-approvals", "/v1/task/approval-history"} {
		h, rec, _ := stack(t, append(approverResults(), detailResults("DEPT_HEAD")...)...)
		code, body := call(t, h, http.MethodGet, path+"?limit=8&page=3", nil, 3001)
		require.Equal(t, http.StatusOK, code, string(body))
		assert.Contains(t, rec.sqls[1], "OFFSET 16 ROWS FETCH NEXT 8 ROWS ONLY", path)

		h, rec, _ = stack(t, append(approverResults(), detailResults("DEPT_HEAD")...)...)
		code, body = call(t, h, http.MethodGet, path+"?limit=1000", nil, 3001)
		require.Equal(t, http.StatusOK, code, string(body))
		assert.Contains(t, rec.sqls[1], "FETCH NEXT 100 ROWS ONLY", path)
	}
}

// status=COMPLETED = "Disetujui" also matches old APPROVED rows (§2.5).
func TestTaskList_StatusCompletedIncludesApproved(t *testing.T) {
	h, rec := setup(t, "USER")

	code, body := get(t, h, "/v1/task?status=COMPLETED")

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Equal(t, `SELECT count(*) FROM "Tasks" WHERE TaskStatus IN ('COMPLETED','APPROVED') AND CreatedBy = 1001`, rec.sqls[1])
	assert.Equal(t, `SELECT * FROM "Tasks" WHERE TaskStatus IN ('COMPLETED','APPROVED') AND CreatedBy = 1001 ORDER BY CreatedAt ASC OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY`, rec.sqls[2])
}
