package repository

import (
	"context"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

// IRoleRepository is read only: Roles is master data kept outside this
// service.
type IRoleRepository interface {
	GetByID(ctx context.Context, roleID string) (role *model.Role, err error)
	// Get returns every role, lowest RoleLevel first.
	Get(ctx context.Context) (roles []*model.Role, err error)
}

type RoleRepository struct {
	opt opt.Options
}

func NewRoleRepository() IRoleRepository {
	return &RoleRepository{
		opt: global.Setup.Common,
	}
}

func (r *RoleRepository) GetByID(ctx context.Context, roleID string) (role *model.Role, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "RoleRepository.GetByID")
	defer span.End()

	db := readDB(ctx, r.opt)

	role = &model.Role{}
	if err = db.Where("RoleID = ?", roleID).First(role).Error; err != nil {
		span.RecordError(err)
		role = nil
		return
	}

	return
}

func (r *RoleRepository) Get(ctx context.Context) (roles []*model.Role, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "RoleRepository.Get")
	defer span.End()

	db := readDB(ctx, r.opt)

	if err = db.Order("RoleLevel ASC").Find(&roles).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}
