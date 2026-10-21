package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
	"gorm.io/gorm"
)

// systemSettingCacheTTL is a week: a setting only changes when an admin
// saves it, and saving refreshes the cache at once.
const systemSettingCacheTTL = 7 * 24 * time.Hour

type ISystemSettingRepository interface {
	// GetByKey reads a setting from the database (no cache); nil when it
	// was never saved.
	GetByKey(ctx context.Context, key string) (setting *model.SystemSetting, err error)
	// GetInt reads a number setting, Redis cache first. Never saved =
	// fallback (not cached).
	GetInt(ctx context.Context, key string, fallback int64) (value int64, err error)
	// UpsertInt saves a number setting and refreshes the cache.
	UpsertInt(ctx context.Context, key string, value int64, updatedBy int64) (err error)
}

type SystemSettingRepository struct {
	opt opt.Options
}

func NewSystemSettingRepository() ISystemSettingRepository {
	return &SystemSettingRepository{
		opt: global.Setup.Common,
	}
}

func (r *SystemSettingRepository) cacheKey(key string) string {
	return redisKey(r.opt, "cache:system_setting:"+key)
}

func (r *SystemSettingRepository) GetByKey(ctx context.Context, key string) (setting *model.SystemSetting, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SystemSettingRepository.GetByKey")
	defer span.End()

	db := readDB(ctx, r.opt)

	setting = &model.SystemSetting{}
	err = db.Where("SettingKey = ?", key).First(setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting, err = nil, nil
		return
	}
	if err != nil {
		err = fmt.Errorf("system_setting: get %s: %w", key, err)
		span.RecordError(err)
		setting = nil
		return
	}

	return
}

func (r *SystemSettingRepository) GetInt(ctx context.Context, key string, fallback int64) (value int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SystemSettingRepository.GetInt")
	defer span.End()

	client, redisErr := redisClient(r.opt, "system_setting")
	if redisErr == nil {
		if cached, getErr := client.Get(ctx, r.cacheKey(key)).Result(); getErr == nil {
			if n, parseErr := strconv.ParseInt(cached, 10, 64); parseErr == nil {
				value = n
				return
			}
		}
	}

	var setting *model.SystemSetting
	setting, err = r.GetByKey(ctx, key)
	if err != nil {
		span.RecordError(err)
		return
	}
	if setting == nil {
		value = fallback
		return
	}

	value, err = strconv.ParseInt(setting.SettingValue, 10, 64)
	if err != nil {
		err = fmt.Errorf("system_setting: %s has non-numeric value %q: %w", key, setting.SettingValue, err)
		span.RecordError(err)
		value = 0
		return
	}

	if redisErr == nil {
		_ = client.Set(ctx, r.cacheKey(key), setting.SettingValue, systemSettingCacheTTL).Err()
	}

	return
}

func (r *SystemSettingRepository) UpsertInt(ctx context.Context, key string, value int64, updatedBy int64) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SystemSettingRepository.UpsertInt")
	defer span.End()

	valueStr := strconv.FormatInt(value, 10)

	var existing *model.SystemSetting
	existing, err = r.GetByKey(ctx, key)
	if err != nil {
		span.RecordError(err)
		return
	}

	db := writeDB(ctx, r.opt)
	if existing == nil {
		row := model.SystemSetting{SettingKey: key, SettingValue: valueStr, UpdatedBy: updatedBy}
		if err = db.Create(&row).Error; err != nil {
			err = fmt.Errorf("system_setting: create %s: %w", key, err)
			span.RecordError(err)
			return
		}
	} else {
		updates := map[string]interface{}{"SettingValue": valueStr, "UpdatedBy": updatedBy}
		if err = db.Model(new(model.SystemSetting)).Where("SettingKey = ?", key).Updates(updates).Error; err != nil {
			err = fmt.Errorf("system_setting: update %s: %w", key, err)
			span.RecordError(err)
			return
		}
	}

	if client, redisErr := redisClient(r.opt, "system_setting"); redisErr == nil {
		_ = client.Set(ctx, r.cacheKey(key), valueStr, systemSettingCacheTTL).Err()
	}

	return
}
