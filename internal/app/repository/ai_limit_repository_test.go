package repository

import (
	"context"
	"testing"
	"time"

	"site-dev-backend/internal/app/constants"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAILimitRepository_TakeReleaseUsed(t *testing.T) {
	o, _ := fakeDBOptions(t)
	srv := fakeRedis(t, &o)
	r := &AILimitRepository{opt: o}
	ctx := context.Background()
	siteKey := "site-dev-api:test:counter:ai_insight_site:-6.230,106.827:"
	totalKey := "site-dev-api:test:counter:ai_insight_total:" + today() + ":"

	for i := int64(1); i <= 3; i++ {
		used, err := r.Take(ctx, constants.AILimitSite, "-6.230,106.827", 3)
		require.NoError(t, err)
		assert.Equal(t, i, used)
	}
	_, err := r.Take(ctx, constants.AILimitSite, "-6.230,106.827", 3)
	assert.ErrorIs(t, err, ErrAILimitReached)
	v, _ := srv.Get(siteKey)
	assert.Equal(t, "3", v, "a refused try is not counted")
	assert.Greater(t, srv.TTL(siteKey), 300*24*time.Hour, "kept for the life of the request")

	require.NoError(t, r.Release(ctx, constants.AILimitSite, "-6.230,106.827"))
	used, err := r.Used(ctx, constants.AILimitSite, "-6.230,106.827")
	require.NoError(t, err)
	assert.Equal(t, int64(2), used)

	// another cell and the day count are separate
	used, err = r.Used(ctx, constants.AILimitSite, "-6.231,106.827")
	require.NoError(t, err)
	assert.Equal(t, int64(0), used)
	_, err = r.Take(ctx, constants.AILimitTotal, "ignored", 60)
	require.NoError(t, err)
	v, _ = srv.Get(totalKey)
	assert.Equal(t, "1", v)
	assert.LessOrEqual(t, srv.TTL(totalKey), 48*time.Hour)

	// give-back of a counter that is gone does not make one
	require.NoError(t, r.Release(ctx, constants.AILimitSite, "-6.299,106.899"))
	_, ok := srv.Get("site-dev-api:test:counter:ai_insight_site:-6.299,106.899:")
	assert.False(t, ok)
}

// The daily AI flush job (counter:ai_insight_quota:*) must not reset the
// site or day counts.
func TestAILimitRepository_NotFlushedByDailyJob(t *testing.T) {
	o, _ := fakeDBOptions(t)
	srv := fakeRedis(t, &o)
	r := &AILimitRepository{opt: o}
	q := &QuotaRepository{opt: o, RepositorySystemSetting: &SystemSettingRepository{opt: o}}
	ctx := context.Background()

	_, err := r.Take(ctx, constants.AILimitSite, "-6.230,106.827", 3)
	require.NoError(t, err)
	_, err = r.Take(ctx, constants.AILimitTotal, "", 60)
	require.NoError(t, err)
	_, err = q.FlushAll(ctx, constants.QuotaAIInsight)
	require.NoError(t, err)
	assert.Len(t, srv.Keys(), 2)
}
