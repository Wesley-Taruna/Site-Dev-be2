package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/constants"

	"github.com/go-redis/redis/v8"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

// Errors when a user has used up today's quota (same texts as site-dev-api).
var (
	ErrAIInsightQuotaExceeded = errors.New("ai_insight: daily generation quota exceeded")
	ErrGeocodingQuotaExceeded = errors.New("geocoding: daily reverse-geocode quota exceeded")
)

const quotaKeyTTL = 24 * time.Hour

// jakartaLoc is the day boundary of the quotas and of the running IDs.
var jakartaLoc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// IQuotaRepository holds the per-user daily counters of the Gemini AI
// insight (constants.QuotaAIInsight) and the Google geocoding
// (constants.QuotaGeocoding) in Redis. A counter starts at the daily limit
// and counts down; tomorrow's key starts fresh.
type IQuotaRepository interface {
	// Limit is the daily limit: SystemSettings, then env.toml, then default.
	Limit(ctx context.Context, kind string) (limit int64, err error)
	// Remaining is what actorID has left today (untouched today = Limit).
	Remaining(ctx context.Context, kind string, actorID int64) (remaining int64, err error)
	// Consume takes one from actorID's counter; below zero =
	// ErrAIInsightQuotaExceeded / ErrGeocodingQuotaExceeded.
	Consume(ctx context.Context, kind string, actorID int64) (remaining int64, err error)
	// FlushUser resets actorID's counter of today.
	FlushUser(ctx context.Context, kind string, actorID int64) (err error)
	// FlushAll deletes every user's counters of kind, any day; returns how
	// many keys were removed.
	FlushAll(ctx context.Context, kind string) (deleted int, err error)
}

type QuotaRepository struct {
	opt                     opt.Options
	RepositorySystemSetting ISystemSettingRepository
}

func NewQuotaRepository() IQuotaRepository {
	return &QuotaRepository{
		opt:                     global.Setup.Common,
		RepositorySystemSetting: NewSystemSettingRepository(),
	}
}

func (r *QuotaRepository) key(kind string, actorID int64) string {
	dateStr := time.Now().In(jakartaLoc).Format("20060102")
	return redisKey(r.opt, fmt.Sprintf("counter:%s_quota:%d:%s", kind, actorID, dateStr))
}

func quotaExceeded(kind string) error {
	if kind == constants.QuotaGeocoding {
		return ErrGeocodingQuotaExceeded
	}
	return ErrAIInsightQuotaExceeded
}

func (r *QuotaRepository) Limit(ctx context.Context, kind string) (limit int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "QuotaRepository.Limit")
	defer span.End()

	fallback := int64(constants.DefaultAIInsightDailyQuota)
	if kind == constants.QuotaGeocoding {
		fallback = constants.DefaultGeocodingDailyQuota
	}
	if r.opt.Env != nil {
		if n := r.opt.Env.GetInt(kind + ".daily_quota"); n > 0 {
			fallback = int64(n)
		}
	}

	limit, err = r.RepositorySystemSetting.GetInt(ctx, constants.QuotaSettingKey(kind), fallback)
	if err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *QuotaRepository) Remaining(ctx context.Context, kind string, actorID int64) (remaining int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "QuotaRepository.Remaining")
	defer span.End()

	client, err := redisClient(r.opt, kind)
	if err != nil {
		span.RecordError(err)
		return
	}

	key := r.key(kind, actorID)
	val, err := client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return r.Limit(ctx, kind)
	}
	if err != nil {
		err = fmt.Errorf("%s: get %s: %w", kind, key, err)
		span.RecordError(err)
		return
	}

	remaining, err = strconv.ParseInt(val, 10, 64)
	if err != nil {
		err = fmt.Errorf("%s: corrupt quota value in redis: %w", kind, err)
		span.RecordError(err)
		remaining = 0
		return
	}
	if remaining < 0 {
		remaining = 0
	}

	return
}

func (r *QuotaRepository) Consume(ctx context.Context, kind string, actorID int64) (remaining int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "QuotaRepository.Consume")
	defer span.End()

	client, err := redisClient(r.opt, kind)
	if err != nil {
		span.RecordError(err)
		return
	}

	limit, err := r.Limit(ctx, kind)
	if err != nil {
		span.RecordError(err)
		return
	}

	// SETNX + DECR: start at the limit, then count down, atomic across
	// concurrent requests
	key := r.key(kind, actorID)
	if err = client.SetNX(ctx, key, limit, quotaKeyTTL).Err(); err != nil {
		err = fmt.Errorf("%s: setnx %s: %w", kind, key, err)
		span.RecordError(err)
		return
	}

	remaining, err = client.Decr(ctx, key).Result()
	if err != nil {
		err = fmt.Errorf("%s: decr %s: %w", kind, key, err)
		span.RecordError(err)
		remaining = 0
		return
	}
	if remaining < 0 {
		err = quotaExceeded(kind)
		span.RecordError(err)
		remaining = 0
		return
	}

	return
}

func (r *QuotaRepository) FlushUser(ctx context.Context, kind string, actorID int64) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "QuotaRepository.FlushUser")
	defer span.End()

	client, err := redisClient(r.opt, kind)
	if err != nil {
		span.RecordError(err)
		return
	}

	key := r.key(kind, actorID)
	if err = client.Del(ctx, key).Err(); err != nil {
		err = fmt.Errorf("%s: del %s: %w", kind, key, err)
		span.RecordError(err)
		return
	}

	return
}

func (r *QuotaRepository) FlushAll(ctx context.Context, kind string) (deleted int, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "QuotaRepository.FlushAll")
	defer span.End()

	client, err := redisClient(r.opt, kind)
	if err != nil {
		span.RecordError(err)
		return
	}

	// SCAN, not KEYS, so a big keyspace does not block Redis
	pattern := redisKey(r.opt, fmt.Sprintf("counter:%s_quota:*", kind))
	var cursor uint64
	for {
		var keys []string
		keys, cursor, err = client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			err = fmt.Errorf("%s: scan %s: %w", kind, pattern, err)
			span.RecordError(err)
			return
		}
		if len(keys) > 0 {
			var n int64
			n, err = client.Del(ctx, keys...).Result()
			if err != nil {
				err = fmt.Errorf("%s: del: %w", kind, err)
				span.RecordError(err)
				return
			}
			deleted += int(n)
		}
		if cursor == 0 {
			break
		}
	}

	return
}
