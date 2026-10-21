package repository

import (
	"context"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type IUserRepository interface {
	GetByID(ctx context.Context, userID int64) (user *model.User, err error)
	// Get returns every user, by username.
	Get(ctx context.Context) (users []*model.User, err error)
	Create(ctx context.Context, user *model.User) (err error)
	// Update changes only the given columns.
	Update(ctx context.Context, userID int64, updates map[string]interface{}) (err error)
	Delete(ctx context.Context, userID int64) (err error)
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

func (r *UserRepository) Get(ctx context.Context) (users []*model.User, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "UserRepository.Get")
	defer span.End()

	db := readDB(ctx, r.opt)

	if err = db.Order("Username ASC").Find(&users).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "UserRepository.Create")
	defer span.End()

	db := writeDB(ctx, r.opt)

	if err = db.Create(user).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *UserRepository) Update(ctx context.Context, userID int64, updates map[string]interface{}) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "UserRepository.Update")
	defer span.End()

	if len(updates) == 0 {
		return
	}

	db := writeDB(ctx, r.opt)

	if err = db.Model(new(model.User)).Where("UserID = ?", userID).Updates(updates).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *UserRepository) Delete(ctx context.Context, userID int64) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "UserRepository.Delete")
	defer span.End()

	db := writeDB(ctx, r.opt)

	if err = db.Where("UserID = ?", userID).Delete(new(model.User)).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}
