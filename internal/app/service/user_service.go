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

type IUserService interface {
	// Get is GET /v1/users: every user with role name and today's quotas.
	Get(ctx context.Context) (res []dto.UserResponseGet, err error)
	Create(ctx context.Context, req dto.UserRequestCreate) (res dto.UserResponseGet, err error)
	Update(ctx context.Context, userID int64, req dto.UserRequestUpdate) (res dto.UserResponseGet, err error)
	Delete(ctx context.Context, userID int64) (err error)
}

type UserService struct {
	opt             opt.Options
	RepositoryUser  repository.IUserRepository
	RepositoryRole  repository.IRoleRepository
	RepositoryQuota repository.IQuotaRepository
}

func NewUserService() IUserService {
	return &UserService{
		opt:             global.Setup.Common,
		RepositoryUser:  repository.NewUserRepository(),
		RepositoryRole:  repository.NewRoleRepository(),
		RepositoryQuota: repository.NewQuotaRepository(),
	}
}

func toUserResponse(m *model.User) dto.UserResponseGet {
	return dto.UserResponseGet{
		UserID:    m.UserID,
		Username:  m.Username,
		Email:     m.Email,
		RoleID:    m.RoleID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func (s *UserService) Get(ctx context.Context) (res []dto.UserResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "UserService.Get")
	defer span.End()

	var users []*model.User
	users, err = s.RepositoryUser.Get(ctx)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	var aiLimit, geoLimit int64
	if aiLimit, err = s.RepositoryQuota.Limit(ctx, constants.QuotaAIInsight); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}
	if geoLimit, err = s.RepositoryQuota.Limit(ctx, constants.QuotaGeocoding); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res = make([]dto.UserResponseGet, 0, len(users))
	for _, u := range users {
		item := toUserResponse(u)
		if role, roleErr := s.RepositoryRole.GetByID(ctx, u.RoleID); roleErr == nil {
			item.RoleName = role.RoleName
		}
		// remaining stays 0 when Redis cannot be read (as in site-dev-api)
		item.AIInsightLimit = aiLimit
		if remaining, quotaErr := s.RepositoryQuota.Remaining(ctx, constants.QuotaAIInsight, u.UserID); quotaErr == nil {
			item.AIInsightRemaining = remaining
		}
		item.GeocodingLimit = geoLimit
		if remaining, quotaErr := s.RepositoryQuota.Remaining(ctx, constants.QuotaGeocoding, u.UserID); quotaErr == nil {
			item.GeocodingRemaining = remaining
		}
		res = append(res, item)
	}

	return
}

func (s *UserService) Create(ctx context.Context, req dto.UserRequestCreate) (res dto.UserResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "UserService.Create")
	defer span.End()

	var role *model.Role
	role, err = s.RepositoryRole.GetByID(ctx, req.RoleID)
	if err != nil {
		err = fmt.Errorf("user_service: role %s not found: %w", req.RoleID, err)
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	user := &model.User{
		UserID:   req.UserID,
		Username: req.Username,
		Email:    req.Email,
		RoleID:   req.RoleID,
	}
	if err = s.RepositoryUser.Create(ctx, user); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res = toUserResponse(user)
	res.RoleName = role.RoleName

	return
}

func (s *UserService) Update(ctx context.Context, userID int64, req dto.UserRequestUpdate) (res dto.UserResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "UserService.Update")
	defer span.End()

	if req.RoleID != "" {
		if _, err = s.RepositoryRole.GetByID(ctx, req.RoleID); err != nil {
			err = fmt.Errorf("user_service: role %s not found: %w", req.RoleID, err)
			span.RecordError(err)
			s.opt.Logger.AddMessage(log.ErrorLevel, err)
			return
		}
	}

	// only the fields that were sent (site-dev-api StructToUpdateMap)
	updates := map[string]interface{}{}
	if req.Username != "" {
		updates["Username"] = req.Username
	}
	if req.Email != "" {
		updates["Email"] = req.Email
	}
	if req.RoleID != "" {
		updates["RoleID"] = req.RoleID
	}
	if err = s.RepositoryUser.Update(ctx, userID, updates); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	var user *model.User
	user, err = s.RepositoryUser.GetByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}
	res = toUserResponse(user)
	if role, roleErr := s.RepositoryRole.GetByID(ctx, user.RoleID); roleErr == nil {
		res.RoleName = role.RoleName
	}

	return
}

func (s *UserService) Delete(ctx context.Context, userID int64) (err error) {
	ctx, span := s.opt.Trace.Start(ctx, "UserService.Delete")
	defer span.End()

	if err = s.RepositoryUser.Delete(ctx, userID); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	return
}
