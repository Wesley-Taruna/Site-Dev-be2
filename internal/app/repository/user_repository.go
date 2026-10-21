package repository

import (
	"context"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type IUserRepository interface {
	GetByID(ctx context.Context, userID int64) (user *model.User, err error)
}

type UserRepository struct {
	opt opt.Options
}

func NewUserRepository() IUserRepository {
	return &UserRepository{
		opt: global.Setup.Common,
	}
}

func (r *UserRepository) GetByID(ctx context.Context, userID int64) (user *model.User, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "UserRepository.GetByID")
	defer span.End()

	db := readDB(ctx, r.opt)

	user = &model.User{}
	if err = db.Where("UserID = ?", userID).First(user).Error; err != nil {
		span.RecordError(err)
		user = nil
		return
	}

	return
}
