package router

// Simple app (Wes, 28 Oct, API-CHANGES-SIMPLE.md): list filters the simple
// version needs on GET /v1/task.

import (
	"net/http"
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
