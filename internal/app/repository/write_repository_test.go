package repository

import (
	"context"
	"fmt"
	"testing"

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

	assert.Equal(t, "ST"+today()+"000001", a)
	assert.Equal(t, "ST"+today()+"000002", b)
	assert.Equal(t, "TS"+today()+"000001", c)
	assert.Equal(t, int64(48*3600), int64(srv.TTL(fmt.Sprintf("site-dev-api:test:counter:site:%s:", today())).Seconds()))
}

func TestApprovalRepositoryUpdateDetailIfPending_RowsAffected(t *testing.T) {
	o, rec := fakeDBOptions(t, sqlfake.Result{Match: `UPDATE "ApprovalDET"`, Affected: sqlfake.Int64(0)})
	r := &ApprovalRepository{opt: o}

	rows, err := r.UpdateDetailIfPending(context.Background(), 31, map[string]interface{}{"Status": "APPROVED"})

	require.NoError(t, err)
	assert.Zero(t, rows)
	assert.Equal(t, `UPDATE "ApprovalDET" SET "Status"='APPROVED',"UpdatedAt"='2026-10-21 08:00:00' WHERE ID = 31 AND Status = 'PENDING'`, rec.sqls[0])
}
