package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/client"
	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type IValidationService interface {
	// ValidateToken is POST /v1/validation: who the caller is. NOTE (plan
	// Q1b): site-dev-api took nik and username from its token; the
	// boilerplate token only has user_id, so nik = user_id and username
	// comes from Users.
	ValidateToken(ctx context.Context, userID int64) (res dto.TokenValidateResponse, err error)
	// ValidateApprovalCode is POST /v1/validation/approval: turns an
	// approval e-mail code into a token, the site to open and the approver.
	ValidateApprovalCode(ctx context.Context, code string) (res dto.ApprovalCodeValidateResponse, err error)
}

type ValidationService struct {
	opt                    opt.Options
	RepositoryUser         repository.IUserRepository
	RepositoryApproval     repository.IApprovalRepository
	RepositoryApprovalCode repository.IApprovalCodeRepository
	RepositoryTask         repository.ITaskRepository
	ClientApprovalJWT      client.IApprovalJWTClient
}

func NewValidationService() IValidationService {
	return &ValidationService{
		opt:                    global.Setup.Common,
		RepositoryUser:         repository.NewUserRepository(),
		RepositoryApproval:     repository.NewApprovalRepository(),
		RepositoryApprovalCode: repository.NewApprovalCodeRepository(),
		RepositoryTask:         repository.NewTaskRepository(),
		ClientApprovalJWT:      global.Setup.Clients.ApprovalJWT,
	}
}

func (s *ValidationService) ValidateToken(ctx context.Context, userID int64) (res dto.TokenValidateResponse, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "ValidationService.ValidateToken")
	defer span.End()

	var user *model.User
	user, err = getUser(ctx, s.RepositoryUser, userID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res = dto.TokenValidateResponse{
		UserID:   user.UserID,
		NIK:      strconv.FormatInt(user.UserID, 10),
		Username: user.Username,
		Role:     user.RoleID,
	}

	return
}

func (s *ValidationService) ValidateApprovalCode(ctx context.Context, code string) (res dto.ApprovalCodeValidateResponse, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "ValidationService.ValidateApprovalCode")
	defer span.End()

	res, err = s.validateApprovalCode(ctx, code)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		res = dto.ApprovalCodeValidateResponse{}
		return
	}

	return
}

func (s *ValidationService) validateApprovalCode(ctx context.Context, code string) (res dto.ApprovalCodeValidateResponse, err error) {
	detailID, err := s.RepositoryApprovalCode.Get(ctx, code)
	if err != nil {
		if errors.Is(err, repository.ErrApprovalCodeNotFound) {
			err = fmt.Errorf("approval_validation: code is invalid or expired")
		}
		return
	}

	detail, err := s.RepositoryApproval.GetDetailByID(ctx, detailID)
	if err != nil {
		err = fmt.Errorf("approval_validation: approval detail not found: %w", err)
		return
	}
	// the code on the row is the real source (Redis could point at a stale row)
	if detail.SecureApprovalCode != code {
		err = fmt.Errorf("approval_validation: code is invalid or expired")
		return
	}
	// only the level that is waiting now; the link stops working once acted on
	if detail.Status != constants.ApprovalDetailStatusPending {
		err = fmt.Errorf("approval_validation: approval is no longer pending")
		return
	}

	header, err := s.RepositoryApproval.GetHeaderByID(ctx, detail.ApprovalHeaderID)
	if err != nil {
		err = fmt.Errorf("approval_validation: approval header not found: %w", err)
		return
	}

	task, err := s.RepositoryTask.GetByID(ctx, header.TaskID)
	if err != nil {
		err = fmt.Errorf("approval_validation: task not found: %w", err)
		return
	}

	approver, err := s.RepositoryUser.GetByID(ctx, detail.ApproverID)
	if err != nil || approver == nil {
		err = fmt.Errorf("approval_validation: approver not found: %w", err)
		return
	}
	nik := strconv.FormatInt(approver.UserID, 10)

	token, err := s.ClientApprovalJWT.IssueJWT(ctx, approver.Username, nik)
	if err != nil {
		return
	}

	res = dto.ApprovalCodeValidateResponse{
		JWT:    token,
		SiteID: task.SiteID,
		UserProfile: dto.ApprovalUserProfile{
			UserID:   approver.UserID,
			NIK:      nik,
			Username: approver.Username,
			Role:     approver.RoleID,
		},
	}
	return
}
