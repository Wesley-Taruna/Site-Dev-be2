package router

// November scope: task_status on the map's request pins (API-CHANGES §4,
// review #6) and status_not on GET /v1/task (review #21; whether the app
// uses it is still open, review §5 #7). New behaviour.

import (
	"database/sql/driver"
	"net/http"
	"testing"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSiteMaps_RequestPinStatus(t *testing.T) {
	h, _, _ := stack(t,
		noRows(`AMFIBI_MASTER.dbo.MStore`),
		sqlfake.Result{Match: `SELECT * FROM "Sites"`, Columns: []string{"SiteID", "SiteName1", "SiteLat", "SiteLon"}, Rows: [][]driver.Value{
			{"ST1", "Toko Satu", -6.2, 106.8}, {"ST2", "Toko Dua", -6.3, 106.9},
		}},
		sqlfake.Result{Match: `SELECT SiteID, TaskStatus FROM "Tasks"`, Columns: []string{"SiteID", "TaskStatus"}, Rows: [][]driver.Value{{"ST1", "HOLD_DIV_HEAD"}}},
	)

	code, body := call(t, h, http.MethodGet, "/v1/site/maps", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	var res struct {
		Sites []map[string]any `json:"sites"`
	}
	require.NoError(t, jsonUnmarshalData(body, &res))
	require.Len(t, res.Sites, 2)
	assert.Equal(t, "HOLD_DIV_HEAD", res.Sites[0]["task_status"])
	assert.NotContains(t, res.Sites[1], "task_status", "no task: left out")
}

func TestTaskList_StatusNot(t *testing.T) {
	h, rec := setup(t, "DEPT_HEAD")

	code, body := get(t, h, "/v1/task?status_not=DRAFT")

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Equal(t, `SELECT count(*) FROM "Tasks" WHERE TaskStatus NOT IN ('DRAFT')`, rec.sqls[1])
	assert.Equal(t, `SELECT * FROM "Tasks" WHERE TaskStatus NOT IN ('DRAFT') ORDER BY CreatedAt ASC OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY`, rec.sqls[2])

	// status wins over status_not
	h, rec = setup(t, "DEPT_HEAD")
	_, _ = get(t, h, "/v1/task?status=REVISION&status_not=DRAFT")
	assert.Equal(t, `SELECT count(*) FROM "Tasks" WHERE TaskStatus = 'REVISION'`, rec.sqls[1])
}
