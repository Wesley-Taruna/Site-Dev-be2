// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// IValidationService is a mock type for the service.IValidationService type
type IValidationService struct {
	mock.Mock
}

// ValidateToken provides a mock function with given fields: ctx, userID
func (_m *IValidationService) ValidateToken(ctx context.Context, userID int64) (dto.TokenValidateResponse, error) {
	ret := _m.Called(ctx, userID)

	if rf, ok := ret.Get(0).(func(context.Context, int64) (dto.TokenValidateResponse, error)); ok {
		return rf(ctx, userID)
	}

	return ret.Get(0).(dto.TokenValidateResponse), ret.Error(1)
}

// ValidateApprovalCode provides a mock function with given fields: ctx, code
func (_m *IValidationService) ValidateApprovalCode(ctx context.Context, code string) (dto.ApprovalCodeValidateResponse, error) {
	ret := _m.Called(ctx, code)

	if rf, ok := ret.Get(0).(func(context.Context, string) (dto.ApprovalCodeValidateResponse, error)); ok {
		return rf(ctx, code)
	}

	return ret.Get(0).(dto.ApprovalCodeValidateResponse), ret.Error(1)
}

// NewIValidationService creates a new instance of IValidationService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIValidationService(t interface {
	mock.TestingT
	Cleanup(func())
}) *IValidationService {
	m := &IValidationService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
