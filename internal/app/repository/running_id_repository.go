package repository

import (
	"context"
	"fmt"
	"time"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/constants"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

const (
	runningNumberWidth = 6
	counterKeyTTL      = 48 * time.Hour
)

// IRunningIDRepository hands out the site and task IDs: prefix + date
// (Asia/Jakarta) + 6-digit running number of that day, from a Redis counter
// (same keys as site-dev-api, so both services never give out the same ID
// as long as app.name/app.env match).
type IRunningIDRepository interface {
	NextSiteID(ctx context.Context) (id string, err error)
	NextTaskID(ctx context.Context) (id string, err error)
}

type RunningIDRepository struct {
	opt opt.Options
}

func NewRunningIDRepository() IRunningIDRepository {
	return &RunningIDRepository{
		opt: global.Setup.Common,
	}
}

func (r *RunningIDRepository) NextSiteID(ctx context.Context) (id string, err error) {
	return r.next(ctx, "site", constants.SiteIDPrefix)
}

func (r *RunningIDRepository) NextTaskID(ctx context.Context) (id string, err error) {
	return r.next(ctx, "task", constants.TaskIDPrefix)
}

func (r *RunningIDRepository) next(ctx context.Context, counterName string, prefix string) (id string, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "RunningIDRepository.Next")
	defer span.End()

	client, err := redisClient(r.opt, "idgen")
	if err != nil {
		span.RecordError(err)
		return
	}

	dateStr := time.Now().In(jakartaLoc).Format("20060102")
	key := redisKey(r.opt, fmt.Sprintf("counter:%s:%s", counterName, dateStr))

	n, err := client.Incr(ctx, key).Result()
	if err != nil {
		err = fmt.Errorf("idgen: incr %s: %w", key, err)
		span.RecordError(err)
		return
	}
	if n == 1 {
		if err = client.Expire(ctx, key, counterKeyTTL).Err(); err != nil {
			err = fmt.Errorf("idgen: expire %s: %w", key, err)
			span.RecordError(err)
			return
		}
	}

	id = fmt.Sprintf("%s%s%0*d", prefix, dateStr, runningNumberWidth, n)
	return
}
