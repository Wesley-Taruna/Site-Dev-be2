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

// IRunningIDRepository hands out the site and task IDs from a daily Redis
// counter per kind (same keys as site-dev-api, so both services never give
// out the same number as long as app.name/app.env match):
//   - task: "TS" + yyyymmdd (Asia/Jakarta) + 6-digit number (unchanged);
//   - site (Wes, 27/28 Oct): "SD" + ddmmyyyy + number of at least 3 digits
//     (constants.FormatSiteID), e.g. "SD28102026001"; the counter starts
//     at 1 every day (WIB).
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
	now := time.Now().In(jakartaLoc)
	n, err := r.incr(ctx, "site", now)
	if err != nil {
		return
	}
	id = constants.FormatSiteID(now, n)
	return
}

func (r *RunningIDRepository) NextTaskID(ctx context.Context) (id string, err error) {
	now := time.Now().In(jakartaLoc)
	n, err := r.incr(ctx, "task", now)
	if err != nil {
		return
	}
	id = fmt.Sprintf("%s%s%0*d", constants.TaskIDPrefix, now.Format("20060102"), runningNumberWidth, n)
	return
}

// incr is the next running number of counterName on day (key per
// yyyymmdd, kept 48 h).
func (r *RunningIDRepository) incr(ctx context.Context, counterName string, day time.Time) (n int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "RunningIDRepository.Next")
	defer span.End()

	client, err := redisClient(r.opt, "idgen")
	if err != nil {
		span.RecordError(err)
		return
	}

	dateStr := day.Format("20060102")
	key := redisKey(r.opt, fmt.Sprintf("counter:%s:%s", counterName, dateStr))

	n, err = client.Incr(ctx, key).Result()
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

	return
}
