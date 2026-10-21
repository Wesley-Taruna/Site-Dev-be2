package service

import (
	"context"
	"fmt"

	"site-dev-backend/global"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type IApprovalMatrixService interface {
	Create(ctx context.Context, req dto.ApprovalMatrixRequestCreate) (res dto.ApprovalMatrixResponseGet, err error)
	Update(ctx context.Context, id uint, req dto.ApprovalMatrixRequestUpdate) (err error)
	// Get is GET /v1/approval-matrix: every row with role and user names.
	Get(ctx context.Context) (res []dto.ApprovalMatrixResponseGet, err error)
}

type ApprovalMatrixService struct {
	opt                      opt.Options
	RepositoryApprovalMatrix repository.IApprovalMatrixRepository
	RepositoryRole           repository.IRoleRepository
	RepositoryUser           repository.IUserRepository
}

func NewApprovalMatrixService() IApprovalMatrixService {
	return &ApprovalMatrixService{
		opt:                      global.Setup.Common,
		RepositoryApprovalMatrix: repository.NewApprovalMatrixRepository(),
		RepositoryRole:           repository.NewRoleRepository(),
		RepositoryUser:           repository.NewUserRepository(),
	}
}

func toApprovalMatrixResponse(m *model.ApprovalMatrix) dto.ApprovalMatrixResponseGet {
	return dto.ApprovalMatrixResponseGet{
		ID:        m.ID,
		RoleID:    m.RoleID,
		UserID:    m.UserID,
		IsActive:  m.IsActive,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func (s *ApprovalMatrixService) Create(ctx context.Context, req dto.ApprovalMatrixRequestCreate) (res dto.ApprovalMatrixResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "ApprovalMatrixService.Create")
	defer span.End()

	var role *model.Role
	role, err = s.RepositoryRole.GetByID(ctx, req.RoleID)
	if err != nil {
		err = fmt.Errorf("approval_matrix_service: role %s not found: %w", req.RoleID, err)
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	var user *model.User
	user, err = s.RepositoryUser.GetByID(ctx, req.UserID)
	if err != nil {
		err = fmt.Errorf("approval_matrix_service: user (nik) %d not found: %w", req.UserID, err)
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	matrix := &model.ApprovalMatrix{
		RoleID:   req.RoleID,
		UserID:   req.UserID,
		IsActive: true,
	}
	if err = s.RepositoryApprovalMatrix.Create(ctx, matrix); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res = toApprovalMatrixResponse(matrix)
	res.RoleName = role.RoleName
	res.RoleLevel = role.RoleLevel
	res.Username = user.Username

	return
}

func (s *ApprovalMatrixService) Update(ctx context.Context, id uint, req dto.ApprovalMatrixRequestUpdate) (err error) {
	ctx, span := s.opt.Trace.Start(ctx, "ApprovalMatrixService.Update")
	defer span.End()

	updates := map[string]interface{}{}
	if req.UserID != 0 {
		updates["UserID"] = req.UserID
	}
	if req.IsActive != nil {
		updates["IsActive"] = *req.IsActive
	}
	if err = s.RepositoryApprovalMatrix.Update(ctx, id, updates); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	return
}

func (s *ApprovalMatrixService) Get(ctx context.Context) (res []dto.ApprovalMatrixResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "ApprovalMatrixService.Get")
	defer span.End()

	var matrices []*model.ApprovalMatrix
	matrices, err = s.RepositoryApprovalMatrix.Get(ctx)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res = make([]dto.ApprovalMatrixResponseGet, 0, len(matrices))
	for _, m := range matrices {
		item := toApprovalMatrixResponse(m)
		if role, roleErr := s.RepositoryRole.GetByID(ctx, m.RoleID); roleErr == nil {
			item.RoleName = role.RoleName
			item.RoleLevel = role.RoleLevel
		}
		if user, userErr := s.RepositoryUser.GetByID(ctx, m.UserID); userErr == nil {
			item.Username = user.Username
		}
		res = append(res, item)
	}

	return
}
