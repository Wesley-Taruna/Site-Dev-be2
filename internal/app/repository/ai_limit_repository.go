package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/constants"

	"github.com/go-redis/redis/v8"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

// ErrAILimitReached: the per-site or server-wide AI analysis limit is
// reached (nothing was counted).
var ErrAILimitReached = errors.New("ai_insight: analysis limit reached")

// aiSiteKeyTTL: a site's count is kept this long after its last analysis
// (the life of a request, resubmits included).
const aiSiteKeyTTL = 365 * 24 * time.Hour

// IAILimitRepository counts AI analyses (Gemini answers) in Redis, on top
// of the per-user daily quota (IQuotaRepository), Wes 23 Oct:
//   - kind constants.AILimitSite: per site location (cell = the rounded
//     point), all users, kept aiSiteKeyTTL;
//   - kind constants.AILimitTotal: all users, per day (Asia/Jakarta); cell
//     is ignored.
//
// The counters count up from 0. Without Redis every call fails (no
// counting = no paid call), like the other quotas.
type IAILimitRepository interface {
	// Take counts one; over limit = ErrAILimitReached and nothing counted.
	Take(ctx context.Context, kind string, cell string, limit int64) (used int64, err error)
	// Release gives one back (the Gemini call failed). A counter that is
	// gone (expired) stays gone.
	Release(ctx context.Context, kind string, cell string) (err error)
	// Used is the count now (none = 0).
	Used(ctx context.Context, kind string, cell string) (used int64, err error)
}

type AILimitRepository struct {
	opt opt.Options
}

func NewAILimitRepository() IAILimitRepository {
	return &AILimitRepository{
		opt: global.Setup.Common,
	}
}

func (r *AILimitRepository) key(kind string, cell string) string {
	if kind == constants.AILimitTotal {
		return redisKey(r.opt, "counter:ai_insight_total:"+time.Now().In(jakartaLoc).Format("20060102"))
	}
	return redisKey(r.opt, "counter:ai_insight_site:"+cell)
}

func (r *AILimitRepository) ttl(kind string) time.Duration {
	if kind == constants.AILimitTotal {
		return 48 * time.Hour
	}
	return aiSiteKeyTTL
}

func (r *AILimitRepository) Take(ctx context.Context, kind string, cell string, limit int64) (used int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "AILimitRepository.Take")
	defer span.End()

	client, err := redisClient(r.opt, "ai_insight limit")
	if err != nil {
		span.RecordError(err)
		return
	}
	key := r.key(kind, cell)
	if used, err = client.Incr(ctx, key).Result(); err != nil {
		err = fmt.Errorf("ai_insight limit: incr %s: %w", key, err)
		span.RecordError(err)
		return
	}
	// a site's count lives on while it is used; the day count from the first
	if used == 1 || kind == constants.AILimitSite {
		_ = client.Expire(ctx, key, r.ttl(kind)).Err()
	}
	if used > limit {
		_ = client.Decr(ctx, key).Err()
		used = limit
		err = ErrAILimitReached
		return
	}

	return
}

func (r *AILimitRepository) Release(ctx context.Context, kind string, cell string) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "AILimitRepository.Release")
	defer span.End()

	client, err := redisClient(r.opt, "ai_insight limit")
	if err != nil {
		span.RecordError(err)
		return
	}
	key := r.key(kind, cell)
	n, err := client.Exists(ctx, key).Result()
	if err != nil || n == 0 {
		return
	}
	if err = client.Decr(ctx, key).Err(); err != nil {
		err = fmt.Errorf("ai_insight limit: decr %s: %w", key, err)
		span.RecordError(err)
		return
	}

	return
}

func (r *AILimitRepository) Used(ctx context.Context, kind string, cell string) (used int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "AILimitRepository.Used")
	defer span.End()

	client, err := redisClient(r.opt, "ai_insight limit")
	if err != nil {
		span.RecordError(err)
		return
	}
	key := r.key(kind, cell)
	used, err = client.Get(ctx, key).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		err = fmt.Errorf("ai_insight limit: get %s: %w", key, err)
		span.RecordError(err)
		return
	}
	if used < 0 {
		used = 0
	}

	return
}
