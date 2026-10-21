package repository

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"site-dev-backend/internal/app/constants"
	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func today() string { return time.Now().In(jakartaLoc).Format("20060102") }

func newQuotaRepo(t *testing.T, results ...sqlfake.Result) (*QuotaRepository, *SystemSettingRepository, func(string) (string, bool)) {
	o, _ := fakeDBOptions(t, results...)
	srv := fakeRedis(t, &o)
	settings := &SystemSettingRepository{opt: o}
	return &QuotaRepository{opt: o, RepositorySystemSetting: settings}, settings, srv.Get
}

func TestQuotaRepositoryConsume_CountsDownThenRefuses(t *testing.T) {
	r, _, get := newQuotaRepo(t, sqlfake.Result{Match: "|ai_insight.daily_quota", Columns: []string{"SettingKey", "SettingValue"}, Rows: [][]driver.Value{{"ai_insight.daily_quota", "2"}}})
	ctx := context.Background()

	n, err := r.Consume(ctx, constants.QuotaAIInsight, 1001)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	n, err = r.Consume(ctx, constants.QuotaAIInsight, 1001)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
	_, err = r.Consume(ctx, constants.QuotaAIInsight, 1001)
	assert.ErrorIs(t, err, ErrAIInsightQuotaExceeded)

	v, ok := get(fmt.Sprintf("site-dev-api:test:counter:ai_insight_quota:1001:%s:", today()))
	assert.True(t, ok)
	assert.Equal(t, "-1", v)
	remaining, err := r.Remaining(ctx, constants.QuotaAIInsight, 1001)
	require.NoError(t, err)
	assert.Equal(t, int64(0), remaining, "below zero is shown as 0")
}

func TestQuotaRepositoryLimit_Fallbacks(t *testing.T) {
	r, _, _ := newQuotaRepo(t)
	ctx := context.Background()

	limit, err := r.Limit(ctx, constants.QuotaGeocoding)
	require.NoError(t, err)
	assert.Equal(t, int64(100), limit, "default")

	r.opt.Env.(interface{ Set(string, any) }).Set("geocoding.daily_quota", 70)
	limit, err = r.Limit(ctx, constants.QuotaGeocoding)
	require.NoError(t, err)
	assert.Equal(t, int64(70), limit, "env.toml")

	remaining, err := r.Remaining(ctx, constants.QuotaGeocoding, 1001)
	require.NoError(t, err)
	assert.Equal(t, int64(70), remaining, "untouched today = full limit")

	_, err = r.Consume(ctx, constants.QuotaGeocoding, 1001)
	require.NoError(t, err)
	for i := 0; i < 69; i++ {
		_, _ = r.Consume(ctx, constants.QuotaGeocoding, 1001)
	}
	_, err = r.Consume(ctx, constants.QuotaGeocoding, 1001)
	assert.ErrorIs(t, err, ErrGeocodingQuotaExceeded)
}

func TestQuotaRepositoryFlush(t *testing.T) {
	r, _, get := newQuotaRepo(t)
	ctx := context.Background()
	_, _ = r.Consume(ctx, constants.QuotaAIInsight, 1)
	_, _ = r.Consume(ctx, constants.QuotaAIInsight, 2)
	_, _ = r.Consume(ctx, constants.QuotaGeocoding, 1)

	require.NoError(t, r.FlushUser(ctx, constants.QuotaAIInsight, 1))
	_, ok := get(fmt.Sprintf("site-dev-api:test:counter:ai_insight_quota:1:%s:", today()))
	assert.False(t, ok)

	n, err := r.FlushAll(ctx, constants.QuotaAIInsight)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "only user 2's AI counter was left")
	_, ok = get(fmt.Sprintf("site-dev-api:test:counter:geocoding_quota:1:%s:", today()))
	assert.True(t, ok, "geocoding counters are untouched")
}

func TestQuotaRepository_NoRedis(t *testing.T) {
	o, _ := fakeDBOptions(t)
	r := &QuotaRepository{opt: o, RepositorySystemSetting: &SystemSettingRepository{opt: o}}

	_, err := r.Consume(context.Background(), constants.QuotaGeocoding, 1)

	assert.EqualError(t, err, "geocoding: redis is not enabled/configured (check [redis] section in config)")
}

func TestSystemSettingRepository_CacheFirst(t *testing.T) {
	_, settings, get := newQuotaRepo(t)
	ctx := context.Background()

	v, err := settings.GetInt(ctx, "ai_insight.daily_quota", 50)
	require.NoError(t, err)
	assert.Equal(t, int64(50), v)
	_, cached := get("site-dev-api:test:cache:system_setting:ai_insight.daily_quota:")
	assert.False(t, cached, "fallback is never cached")

	require.NoError(t, settings.UpsertInt(ctx, "ai_insight.daily_quota", 12, 9001))
	v, err = settings.GetInt(ctx, "ai_insight.daily_quota", 50)
	require.NoError(t, err)
	assert.Equal(t, int64(12), v, "read from the cache written by UpsertInt")
}

func TestSystemSettingRepositoryUpsertInt_UpdatesExisting(t *testing.T) {
	o, rec := fakeDBOptions(t, sqlfake.Result{Match: "|geocoding.daily_quota", Columns: []string{"ID", "SettingKey", "SettingValue"}, Rows: [][]driver.Value{{int64(2), "geocoding.daily_quota", "100"}}})
	r := &SystemSettingRepository{opt: o}

	require.NoError(t, r.UpsertInt(context.Background(), "geocoding.daily_quota", 250, 9001))

	assert.Equal(t, `UPDATE "SystemSettings" SET "SettingValue"='250',"UpdatedBy"=9001,"UpdatedAt"='2026-10-21 08:00:00' WHERE SettingKey = 'geocoding.daily_quota'`, rec.sqls[1])
}
