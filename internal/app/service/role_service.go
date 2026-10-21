package service

import (
	"context"

	"site-dev-backend/global"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type IRoleService interface {
	// Get is GET /v1/roles: every role, lowest level first.
	Get(ctx context.Context) (res []dto.RoleResponseGet, err error)
}

type RoleService struct {
	opt            opt.Options
	RepositoryRole repository.IRoleRepository
}

func NewRoleService() IRoleService {
	return &RoleService{
		opt:            global.Setup.Common,
		RepositoryRole: repository.NewRoleRepository(),
	}
}

func (s *RoleService) Get(ctx context.Context) (res []dto.RoleResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "RoleService.Get")
	defer span.End()

	var roles []*model.Role
	roles, err = s.RepositoryRole.Get(ctx)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res = make([]dto.RoleResponseGet, 0, len(roles))
	for _, role := range roles {
		res = append(res, dto.RoleResponseGet{RoleID: role.RoleID, RoleName: role.RoleName, RoleLevel: role.RoleLevel})
	}

	return
}
