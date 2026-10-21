package repository

import (
	"context"
	"fmt"
	"time"

	"site-dev-backend/global"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

// ErrNearbyDailyCap: the server-wide daily cap of Google Places calls is
// reached.
var ErrNearbyDailyCap = fmt.Errorf("places: daily Google call cap reached")

// INearbyAreaRepository keeps fetched nearby-place areas (JSON, per ~110 m
// cell) and the server-wide daily count of Google Places calls in Redis
// (local §6). Without Redis nothing is cached and Google is not used.
type INearbyAreaRepository interface {
	GetArea(ctx context.Context, key string) (raw []byte, found bool)
	SetArea(ctx context.Context, key string, raw []byte, ttl time.Duration)
	// TakeGoogleCall counts one call of today; over dailyCap =
	// ErrNearbyDailyCap.
	TakeGoogleCall(ctx context.Context, dailyCap int64) (err error)
	// GoogleCallsLeft is what is left of dailyCap today.
	GoogleCallsLeft(ctx context.Context, dailyCap int64) (left int64)
}

type NearbyAreaRepository struct {
	opt opt.Options
}

func NewNearbyAreaRepository() INearbyAreaRepository {
	return &NearbyAreaRepository{
		opt: global.Setup.Common,
	}
}

func (r *NearbyAreaRepository) areaKey(key string) string {
	return redisKey(r.opt, "cache:nearby_area:"+key)
}

// budgetKey follows the quota keys (Asia/Jakarta date), server wide.
func (r *NearbyAreaRepository) budgetKey() string {
	return redisKey(r.opt, "counter:places_google_calls:"+time.Now().In(jakartaLoc).Format("20060102"))
}

func (r *NearbyAreaRepository) GetArea(ctx context.Context, key string) (raw []byte, found bool) {
	ctx, span := r.opt.Trace.Start(ctx, "NearbyAreaRepository.GetArea")
	defer span.End()

	client, err := redisClient(r.opt, "places")
	if err != nil {
		return nil, false
	}
	raw, err = client.Get(ctx, r.areaKey(key)).Bytes()
	if err != nil {
		return nil, false
	}
	return raw, true
}

func (r *NearbyAreaRepository) SetArea(ctx context.Context, key string, raw []byte, ttl time.Duration) {
	ctx, span := r.opt.Trace.Start(ctx, "NearbyAreaRepository.SetArea")
	defer span.End()

	client, err := redisClient(r.opt, "places")
	if err != nil {
		return
	}
	if err = client.Set(ctx, r.areaKey(key), raw, ttl).Err(); err != nil {
		span.RecordError(err)
	}
}

func (r *NearbyAreaRepository) TakeGoogleCall(ctx context.Context, dailyCap int64) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "NearbyAreaRepository.TakeGoogleCall")
	defer span.End()

	client, err := redisClient(r.opt, "places")
	if err != nil {
		span.RecordError(err)
		return
	}
	key := r.budgetKey()
	n, err := client.Incr(ctx, key).Result()
	if err != nil {
		err = fmt.Errorf("places budget: %w", err)
		span.RecordError(err)
		return
	}
	if n == 1 {
		_ = client.Expire(ctx, key, 48*time.Hour).Err()
	}
	if n > dailyCap {
		err = ErrNearbyDailyCap
		return
	}

	return
}

func (r *NearbyAreaRepository) GoogleCallsLeft(ctx context.Context, dailyCap int64) (left int64) {
	ctx, span := r.opt.Trace.Start(ctx, "NearbyAreaRepository.GoogleCallsLeft")
	defer span.End()

	client, err := redisClient(r.opt, "places")
	if err != nil {
		return 0
	}
	n, err := client.Get(ctx, r.budgetKey()).Int64()
	if err != nil {
		n = 0 // no key = nothing used today
	}
	if n >= dailyCap {
		return 0
	}
	return dailyCap - n
}
