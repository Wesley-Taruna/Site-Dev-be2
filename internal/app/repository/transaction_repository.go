package repository

import (
	"context"
	"fmt"

	"site-dev-backend/global"

	"github.com/go-redis/redis/v8"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
	"gitlab.com/fami-app-v2/familib/familabs/redisx"
	"gorm.io/gorm"
)

// ITransactionRepository runs several repository calls in one database
// transaction: every repository method called with the ctx that fn
// receives uses that transaction (same as site-dev-api's
// repository.WithTransaction).
type ITransactionRepository interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) (err error)
}

type TransactionRepository struct {
	opt opt.Options
}

func NewTransactionRepository() ITransactionRepository {
	return &TransactionRepository{
		opt: global.Setup.Common,
	}
}

type txKey struct{}

func (r *TransactionRepository) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TransactionRepository.Do")
	defer span.End()

	err = r.opt.Database.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
	if err != nil {
		span.RecordError(err)
		return
	}

	return
}

// readDB is opt.Database.Read, or the open transaction of ctx.
func readDB(ctx context.Context, o opt.Options) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return o.Database.Read.WithContext(ctx)
}

// writeDB is opt.Database.Write, or the open transaction of ctx.
func writeDB(ctx context.Context, o opt.Options) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return o.Database.Write.WithContext(ctx)
}

// redisClient returns the go-redis client behind opt.Redisx; scope is the
// prefix of the error message (same messages as site-dev-api).
func redisClient(o opt.Options, scope string) (*redis.Client, error) {
	rx, ok := o.Redisx.(*redisx.Redisx)
	if !ok || rx == nil || rx.Client == nil {
		return nil, fmt.Errorf("%s: redis is not enabled/configured (check [redis] section in config)", scope)
	}
	return rx.Client, nil
}

// redisKey prefixes a key with "<app.name>:<app.env>:" like site-dev-api's
// global.RedisPrefix, so both services read and write the same keys as long
// as app.name/app.env in the config are the same.
func redisKey(o opt.Options, key string) string {
	return fmt.Sprintf("%s:%s:%s:", o.Config.App.Name, o.Config.App.Env, key)
}
