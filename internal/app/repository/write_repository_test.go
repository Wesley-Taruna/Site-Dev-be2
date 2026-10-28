package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunningIDRepository_DailyCounters(t *testing.T) {
	o, _ := fakeDBOptions(t)
	srv := fakeRedis(t, &o)
	r := &RunningIDRepository{opt: o}
	ctx := context.Background()

	a, err := r.NextSiteID(ctx)
	require.NoError(t, err)
	b, _ := r.NextSiteID(ctx)
	c, _ := r.NextTaskID(ctx)

	// site ID (Wes, 28 Oct): SD + ddmmyyyy + 3 digits, no slashes; task ID unchanged
	day := time.Now().In(jakartaLoc).Format("02012006")
	assert.Equal(t, "SD"+day+"001", a)
	assert.Equal(t, "SD"+day+"002", b)
	assert.Equal(t, "TS"+today()+"000001", c)
	assert.Equal(t, int64(48*3600), int64(srv.TTL(fmt.Sprintf("site-dev-api:test:counter:site:%s:", today())).Seconds()))
}

// The site number grows past 3 digits instead of wrapping (1000th site of a
// day = 4 digits).
func TestRunningIDRepository_SiteIDFourDigits(t *testing.T) {
	o, _ := fakeDBOptions(t)
	srv := fakeRedis(t, &o)
	r := &RunningIDRepository{opt: o}
	srv.Set(fmt.Sprintf("site-dev-api:test:counter:site:%s:", today()), "999")

	id, err := r.NextSiteID(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "SD"+time.Now().In(jakartaLoc).Format("02012006")+"1000", id)
	assert.Len(t, id, 14, "fits Sites.SiteID varchar(20)")
	assert.NotContains(t, id, "/", "URL-safe as is")
}

func TestApprovalRepositoryUpdateDetailIfPending_RowsAffected(t *testing.T) {
	o, rec := fakeDBOptions(t, sqlfake.Result{Match: `UPDATE "ApprovalDET"`, Affected: sqlfake.Int64(0)})
	r := &ApprovalRepository{opt: o}

	rows, err := r.UpdateDetailIfPending(context.Background(), 31, map[string]interface{}{"Status": "APPROVED"})

	require.NoError(t, err)
	assert.Zero(t, rows)
	assert.Equal(t, `UPDATE "ApprovalDET" SET "Status"='APPROVED',"UpdatedAt"='2026-10-21 08:00:00' WHERE ID = 31 AND Status = 'PENDING'`, rec.sqls[0])
}
