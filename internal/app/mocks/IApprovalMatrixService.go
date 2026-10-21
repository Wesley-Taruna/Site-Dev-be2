// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// IApprovalMatrixService is a mock type for the service.IApprovalMatrixService type
type IApprovalMatrixService struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx, req
func (_m *IApprovalMatrixService) Create(ctx context.Context, req dto.ApprovalMatrixRequestCreate) (dto.ApprovalMatrixResponseGet, error) {
	ret := _m.Called(ctx, req)

	if rf, ok := ret.Get(0).(func(context.Context, dto.ApprovalMatrixRequestCreate) (dto.ApprovalMatrixResponseGet, error)); ok {
		return rf(ctx, req)
	}

	return ret.Get(0).(dto.ApprovalMatrixResponseGet), ret.Error(1)
}

// Update provides a mock function with given fields: ctx, id, req
func (_m *IApprovalMatrixService) Update(ctx context.Context, id uint, req dto.ApprovalMatrixRequestUpdate) error {
	ret := _m.Called(ctx, id, req)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, dto.ApprovalMatrixRequestUpdate) error); ok {
		r0 = rf(ctx, id, req)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Get provides a mock function with given fields: ctx
func (_m *IApprovalMatrixService) Get(ctx context.Context) ([]dto.ApprovalMatrixResponseGet, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) ([]dto.ApprovalMatrixResponseGet, error)); ok {
		return rf(ctx)
	}

	var r0 []dto.ApprovalMatrixResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.ApprovalMatrixResponseGet)
	}
	return r0, ret.Error(1)
}

// NewIApprovalMatrixService creates a new instance of IApprovalMatrixService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIApprovalMatrixService(t interface {
	mock.TestingT
	Cleanup(func())
}) *IApprovalMatrixService {
	m := &IApprovalMatrixService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
