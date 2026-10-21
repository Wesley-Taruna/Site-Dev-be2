// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// IApprovalMatrixRepository is a mock type for the repository.IApprovalMatrixRepository type
type IApprovalMatrixRepository struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx, matrix
func (_m *IApprovalMatrixRepository) Create(ctx context.Context, matrix *model.ApprovalMatrix) error {
	ret := _m.Called(ctx, matrix)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *model.ApprovalMatrix) error); ok {
		r0 = rf(ctx, matrix)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Update provides a mock function with given fields: ctx, id, updates
func (_m *IApprovalMatrixRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	ret := _m.Called(ctx, id, updates)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, map[string]interface{}) error); ok {
		r0 = rf(ctx, id, updates)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Get provides a mock function with given fields: ctx
func (_m *IApprovalMatrixRepository) Get(ctx context.Context) ([]*model.ApprovalMatrix, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) ([]*model.ApprovalMatrix, error)); ok {
		return rf(ctx)
	}

	var r0 []*model.ApprovalMatrix
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.ApprovalMatrix)
	}
	return r0, ret.Error(1)
}

// GetActiveChain provides a mock function with given fields: ctx
func (_m *IApprovalMatrixRepository) GetActiveChain(ctx context.Context) ([]model.ApprovalChainStep, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) ([]model.ApprovalChainStep, error)); ok {
		return rf(ctx)
	}

	var r0 []model.ApprovalChainStep
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.ApprovalChainStep)
	}
	return r0, ret.Error(1)
}

// NewIApprovalMatrixRepository creates a new instance of IApprovalMatrixRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIApprovalMatrixRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *IApprovalMatrixRepository {
	m := &IApprovalMatrixRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
