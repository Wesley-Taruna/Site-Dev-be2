package repository

import (
	"context"
	"errors"
	"testing"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionRepositoryDo_CommitsAndSharesTheTransaction(t *testing.T) {
	o, rec, fdb := fakeDB(t)
	tx := &TransactionRepository{opt: o}
	users := &UserRepository{opt: o}

	err := tx.Do(context.Background(), func(ctx context.Context) error {
		_, _ = users.GetByID(ctx, 1001)
		return writeDB(ctx, o).Model(&model.Task{}).Where("TaskID = ?", "TS1").Update("TaskStatus", "DRAFT").Error
	})

	require.NoError(t, err)
	assert.Equal(t, []string{
		"SELECT * FROM \"Users\" WHERE UserID = 1001 ORDER BY \"Users\".\"UserID\" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY",
		"UPDATE \"Tasks\" SET \"TaskStatus\"='DRAFT',\"UpdatedAt\"='2026-10-21 08:00:00' WHERE TaskID = 'TS1'",
	}, rec.sqls)
	q := fdb.Queries()
	assert.Equal(t, "BEGIN", q[0])
	assert.Equal(t, "COMMIT", q[len(q)-1])
	assert.NotContains(t, q, "ROLLBACK")
}

func TestTransactionRepositoryDo_RollsBackOnError(t *testing.T) {
	o, _, fdb := fakeDB(t)
	tx := &TransactionRepository{opt: o}
	boom := errors.New("boom")

	err := tx.Do(context.Background(), func(ctx context.Context) error { return boom })

	assert.ErrorIs(t, err, boom)
	assert.Equal(t, []string{"BEGIN", "ROLLBACK"}, fdb.Queries())
}

func TestRedisKey_SameAsSiteDevAPI(t *testing.T) {
	o, _, _ := fakeDB(t)
	// site-dev-api global.RedisPrefix: "<app.name>:<app.env>:<prefix>:"
	assert.Equal(t, "site-dev-api:test:counter:task:20261021:", redisKey(o, "counter:task:20261021"))
}

func TestRedisFake_RoundTrip(t *testing.T) {
	o, _, _ := fakeDB(t)
	srv := fakeRedis(t, &o)
	c, err := redisClient(o, "test")
	require.NoError(t, err)
	ctx := context.Background()

	n, err := c.Incr(ctx, "k").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	ok, err := c.SetNX(ctx, "q", 5, 0).Result()
	require.NoError(t, err)
	assert.True(t, ok)
	n, err = c.Decr(ctx, "q").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(4), n)
	keys, cur, err := c.Scan(ctx, 0, "*", 100).Result()
	require.NoError(t, err)
	assert.Equal(t, uint64(0), cur)
	assert.Equal(t, []string{"k", "q"}, keys)
	assert.Equal(t, []string{"k", "q"}, srv.Keys())
}

func TestRedisClient_NotConfigured(t *testing.T) {
	o, _, _ := fakeDB(t)
	_, err := redisClient(o, "idgen")
	assert.EqualError(t, err, "idgen: redis is not enabled/configured (check [redis] section in config)")
}
