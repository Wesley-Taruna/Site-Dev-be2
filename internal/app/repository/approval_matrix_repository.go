package repository

import (
	"context"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type IApprovalMatrixRepository interface {
	Create(ctx context.Context, matrix *model.ApprovalMatrix) (err error)
	// Update changes only the given columns.
	Update(ctx context.Context, id uint, updates map[string]interface{}) (err error)
	// Get returns every row, by RoleID.
	Get(ctx context.Context) (matrices []*model.ApprovalMatrix, err error)
	// GetActiveChain returns the active rows with their Roles.RoleLevel,
	// lowest level first: the approval order.
	GetActiveChain(ctx context.Context) (steps []model.ApprovalChainStep, err error)
}

type ApprovalMatrixRepository struct {
	opt opt.Options
}

func NewApprovalMatrixRepository() IApprovalMatrixRepository {
	return &ApprovalMatrixRepository{
		opt: global.Setup.Common,
	}
}

func (r *ApprovalMatrixRepository) Create(ctx context.Context, matrix *model.ApprovalMatrix) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalMatrixRepository.Create")
	defer span.End()

	db := writeDB(ctx, r.opt)

	if err = db.Create(matrix).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *ApprovalMatrixRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalMatrixRepository.Update")
	defer span.End()

	if len(updates) == 0 {
		return
	}

	db := writeDB(ctx, r.opt)

	if err = db.Model(new(model.ApprovalMatrix)).Where("ID = ?", id).Updates(updates).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *ApprovalMatrixRepository) Get(ctx context.Context) (matrices []*model.ApprovalMatrix, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalMatrixRepository.Get")
	defer span.End()

	db := readDB(ctx, r.opt)

	if err = db.Order("RoleID ASC").Find(&matrices).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *ApprovalMatrixRepository) GetActiveChain(ctx context.Context) (steps []model.ApprovalChainStep, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalMatrixRepository.GetActiveChain")
	defer span.End()

	db := readDB(ctx, r.opt)

	err = db.Table("ApprovalMatrix").
		Select("ApprovalMatrix.RoleID, Roles.RoleLevel, ApprovalMatrix.UserID").
		Joins("JOIN Roles ON Roles.RoleID = ApprovalMatrix.RoleID").
		Where("ApprovalMatrix.IsActive = ?", true).
		Order("Roles.RoleLevel ASC").
		Scan(&steps).Error
	if err != nil {
		span.RecordError(err)
		return
	}

	return
}
