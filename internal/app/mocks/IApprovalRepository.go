// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// IApprovalRepository is a mock type for the repository.IApprovalRepository type
type IApprovalRepository struct {
	mock.Mock
}

// GetHeaderByTaskID provides a mock function with given fields: ctx, taskID
func (_m *IApprovalRepository) GetHeaderByTaskID(ctx context.Context, taskID string) (*model.ApprovalHeader, error) {
	ret := _m.Called(ctx, taskID)

	if rf, ok := ret.Get(0).(func(context.Context, string) (*model.ApprovalHeader, error)); ok {
		return rf(ctx, taskID)
	}

	var r0 *model.ApprovalHeader
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.ApprovalHeader)
	}
	return r0, ret.Error(1)
}

// GetHeaderByID provides a mock function with given fields: ctx, id
func (_m *IApprovalRepository) GetHeaderByID(ctx context.Context, id uint) (*model.ApprovalHeader, error) {
	ret := _m.Called(ctx, id)

	if rf, ok := ret.Get(0).(func(context.Context, uint) (*model.ApprovalHeader, error)); ok {
		return rf(ctx, id)
	}

	var r0 *model.ApprovalHeader
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.ApprovalHeader)
	}
	return r0, ret.Error(1)
}

// GetDetailsByHeaderID provides a mock function with given fields: ctx, headerID
func (_m *IApprovalRepository) GetDetailsByHeaderID(ctx context.Context, headerID uint) ([]*model.ApprovalDetail, error) {
	ret := _m.Called(ctx, headerID)

	if rf, ok := ret.Get(0).(func(context.Context, uint) ([]*model.ApprovalDetail, error)); ok {
		return rf(ctx, headerID)
	}

	var r0 []*model.ApprovalDetail
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.ApprovalDetail)
	}
	return r0, ret.Error(1)
}

// GetPendingByApprover provides a mock function with given fields: ctx, approverID, search, orderBy, limit, offset
func (_m *IApprovalRepository) GetPendingByApprover(ctx context.Context, approverID int64, search string, orderBy string, limit int, offset int) ([]*model.ApprovalDetail, int64, error) {
	ret := _m.Called(ctx, approverID, search, orderBy, limit, offset)

	if rf, ok := ret.Get(0).(func(context.Context, int64, string, string, int, int) ([]*model.ApprovalDetail, int64, error)); ok {
		return rf(ctx, approverID, search, orderBy, limit, offset)
	}

	var r0 []*model.ApprovalDetail
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.ApprovalDetail)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetActedByApprover provides a mock function with given fields: ctx, approverID, search, orderBy, limit, offset
func (_m *IApprovalRepository) GetActedByApprover(ctx context.Context, approverID int64, search string, orderBy string, limit int, offset int) ([]*model.ApprovalDetail, int64, error) {
	ret := _m.Called(ctx, approverID, search, orderBy, limit, offset)

	if rf, ok := ret.Get(0).(func(context.Context, int64, string, string, int, int) ([]*model.ApprovalDetail, int64, error)); ok {
		return rf(ctx, approverID, search, orderBy, limit, offset)
	}

	var r0 []*model.ApprovalDetail
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.ApprovalDetail)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// NewIApprovalRepository creates a new instance of IApprovalRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIApprovalRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *IApprovalRepository {
	m := &IApprovalRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
