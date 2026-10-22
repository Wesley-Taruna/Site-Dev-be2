package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"site-dev-backend/global"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

// ErrGoogleDailyCap: the server-wide daily cap of one Google API is
// reached.
var ErrGoogleDailyCap = errors.New("google: server-wide daily call cap reached")

// IGoogleBudgetRepository counts server-wide Google calls per API and day
// (Asia/Jakarta) in Redis, on top of the per-user quotas. Without Redis
// the call is refused (no counting = no paid call).
type IGoogleBudgetRepository interface {
	// Take counts one call of api today; over dailyCap = ErrGoogleDailyCap.
	Take(ctx context.Context, api string, dailyCap int64) (err error)
}

type GoogleBudgetRepository struct {
	opt opt.Options
}

func NewGoogleBudgetRepository() IGoogleBudgetRepository {
	return &GoogleBudgetRepository{
		opt: global.Setup.Common,
	}
}

func (r *GoogleBudgetRepository) key(api string) string {
	return redisKey(r.opt, "counter:google_calls:"+api+":"+time.Now().In(jakartaLoc).Format("20060102"))
}

func (r *GoogleBudgetRepository) Take(ctx context.Context, api string, dailyCap int64) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "GoogleBudgetRepository.Take")
	defer span.End()

	client, err := redisClient(r.opt, "google budget")
	if err != nil {
		span.RecordError(err)
		return
	}
	key := r.key(api)
	n, err := client.Incr(ctx, key).Result()
	if err != nil {
		err = fmt.Errorf("google budget: %w", err)
		span.RecordError(err)
		return
	}
	if n == 1 {
		_ = client.Expire(ctx, key, 48*time.Hour).Err()
	}
	if n > dailyCap {
		err = ErrGoogleDailyCap
		return
	}

	return
}
