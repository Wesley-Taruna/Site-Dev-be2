package service

import (
	"context"
	"fmt"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type ISettingsService interface {
	// GetLimits is GET /v1/settings/limits: both daily limits and who
	// changed them last.
	GetLimits(ctx context.Context) (res dto.LimitsResponseGet, err error)
	UpdateLimits(ctx context.Context, req dto.LimitsRequestUpdate, updatedBy int64) (res dto.LimitsResponseGet, err error)
	// FlushUserQuota resets one user's quota of today ("Reset Limit
	// Pengguna").
	FlushUserQuota(ctx context.Context, req dto.QuotaFlushRequest) (res dto.QuotaFlushResponse, err error)
}

type SettingsService struct {
	opt                     opt.Options
	RepositoryUser          repository.IUserRepository
	RepositorySystemSetting repository.ISystemSettingRepository
	RepositoryQuota         repository.IQuotaRepository
}

func NewSettingsService() ISettingsService {
	return &SettingsService{
		opt:                     global.Setup.Common,
		RepositoryUser:          repository.NewUserRepository(),
		RepositorySystemSetting: repository.NewSystemSettingRepository(),
		RepositoryQuota:         repository.NewQuotaRepository(),
	}
}

func (s *SettingsService) GetLimits(ctx context.Context) (res dto.LimitsResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "SettingsService.GetLimits")
	defer span.End()

	res, err = s.limits(ctx)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	return
}

func (s *SettingsService) limits(ctx context.Context) (res dto.LimitsResponseGet, err error) {
	if res.AIInsightDailyQuota, err = s.RepositoryQuota.Limit(ctx, constants.QuotaAIInsight); err != nil {
		return dto.LimitsResponseGet{}, err
	}
	if res.GeocodingDailyQuota, err = s.RepositoryQuota.Limit(ctx, constants.QuotaGeocoding); err != nil {
		return dto.LimitsResponseGet{}, err
	}

	// "last changed by" is best effort, read without cache
	var setting *model.SystemSetting
	if setting, _ = s.RepositorySystemSetting.GetByKey(ctx, constants.QuotaSettingKey(constants.QuotaAIInsight)); setting != nil {
		res.AIInsightUpdatedBy = setting.UpdatedBy
		updatedAt := setting.UpdatedAt
		res.AIInsightUpdatedAt = &updatedAt
		if u, userErr := s.RepositoryUser.GetByID(ctx, setting.UpdatedBy); userErr == nil {
			res.AIInsightUpdatedByName = u.Username
		}
	}
	if setting, _ = s.RepositorySystemSetting.GetByKey(ctx, constants.QuotaSettingKey(constants.QuotaGeocoding)); setting != nil {
		res.GeocodingUpdatedBy = setting.UpdatedBy
		updatedAt := setting.UpdatedAt
		res.GeocodingUpdatedAt = &updatedAt
		if u, userErr := s.RepositoryUser.GetByID(ctx, setting.UpdatedBy); userErr == nil {
			res.GeocodingUpdatedByName = u.Username
		}
	}
	return res, nil
}

func (s *SettingsService) UpdateLimits(ctx context.Context, req dto.LimitsRequestUpdate, updatedBy int64) (res dto.LimitsResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "SettingsService.UpdateLimits")
	defer span.End()

	if req.AIInsightDailyQuota != nil {
		if err = s.RepositorySystemSetting.UpsertInt(ctx, constants.QuotaSettingKey(constants.QuotaAIInsight), *req.AIInsightDailyQuota, updatedBy); err != nil {
			span.RecordError(err)
			s.opt.Logger.AddMessage(log.ErrorLevel, err)
			return
		}
	}
	if req.GeocodingDailyQuota != nil {
		if err = s.RepositorySystemSetting.UpsertInt(ctx, constants.QuotaSettingKey(constants.QuotaGeocoding), *req.GeocodingDailyQuota, updatedBy); err != nil {
			span.RecordError(err)
			s.opt.Logger.AddMessage(log.ErrorLevel, err)
			return
		}
	}

	res, err = s.limits(ctx)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	return
}

func (s *SettingsService) FlushUserQuota(ctx context.Context, req dto.QuotaFlushRequest) (res dto.QuotaFlushResponse, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "SettingsService.FlushUserQuota")
	defer span.End()

	if _, err = s.RepositoryUser.GetByID(ctx, req.UserID); err != nil {
		err = fmt.Errorf("settings_service: user %d not found: %w", req.UserID, err)
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	if req.QuotaType != constants.QuotaAIInsight && req.QuotaType != constants.QuotaGeocoding {
		err = fmt.Errorf("settings_service: unknown quota_type %q", req.QuotaType)
		span.RecordError(err)
		return
	}

	if err = s.RepositoryQuota.FlushUser(ctx, req.QuotaType, req.UserID); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	var limit int64
	limit, err = s.RepositoryQuota.Limit(ctx, req.QuotaType)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}
	res = dto.QuotaFlushResponse{UserID: req.UserID, QuotaType: req.QuotaType, Remaining: limit, Limit: limit}

	return
}
