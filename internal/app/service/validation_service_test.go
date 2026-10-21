package service

import (
	"context"
	"testing"

	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestValidationService(t *testing.T) (*ValidationService, *mocks.IApprovalCodeRepository, *mocks.IApprovalRepository, *mocks.IApprovalJWTClient) {
	codes, approvals, jwt := mocks.NewIApprovalCodeRepository(t), mocks.NewIApprovalRepository(t), mocks.NewIApprovalJWTClient(t)
	return &ValidationService{opt: testOptions(), RepositoryApprovalCode: codes, RepositoryApproval: approvals, ClientApprovalJWT: jwt,
		RepositoryUser: mocks.NewIUserRepository(t), RepositoryTask: mocks.NewITaskRepository(t)}, codes, approvals, jwt
}

func TestValidationService_CodeOnRowMustMatch(t *testing.T) {
	s, codes, approvals, jwt := newTestValidationService(t)
	codes.On("Get", ctxAny, "A").Return(uint(32), nil)
	approvals.On("GetDetailByID", ctxAny, uint(32)).Return(&model.ApprovalDetail{ID: 32, SecureApprovalCode: "B", Status: "PENDING"}, nil)

	_, err := s.ValidateApprovalCode(context.Background(), "A")

	assert.EqualError(t, err, "approval_validation: code is invalid or expired")
	jwt.AssertNotCalled(t, "IssueJWT")
}

func TestValidationService_RedisDownIsReturned(t *testing.T) {
	s, codes, _, _ := newTestValidationService(t)
	codes.On("Get", ctxAny, "A").Return(uint(0), assert.AnError)

	_, err := s.ValidateApprovalCode(context.Background(), "A")

	assert.ErrorIs(t, err, assert.AnError)
}

func TestValidationService_NotFoundCode(t *testing.T) {
	s, codes, _, _ := newTestValidationService(t)
	codes.On("Get", ctxAny, "A").Return(uint(0), repository.ErrApprovalCodeNotFound)

	_, err := s.ValidateApprovalCode(context.Background(), "A")

	require.Error(t, err)
	assert.Equal(t, "approval_validation: code is invalid or expired", err.Error())
}

func TestGenerateSecureApprovalCode(t *testing.T) {
	a, err := repository.GenerateSecureApprovalCode()
	require.NoError(t, err)
	b, _ := repository.GenerateSecureApprovalCode()
	assert.Len(t, a, 32)
	assert.NotEqual(t, a, b)
	assert.Regexp(t, `^[A-Za-z0-9]{32}$`, a)
}
