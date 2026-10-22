// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// IActivityService is a mock type for the service.IActivityService type
type IActivityService struct {
	mock.Mock
}

// Get provides a mock function with given fields: ctx, req, userID
func (_m *IActivityService) Get(ctx context.Context, req dto.ActivityRequestGet, userID int64) ([]dto.ActivityResponseGet, int64, error) {
	ret := _m.Called(ctx, req, userID)

	if rf, ok := ret.Get(0).(func(context.Context, dto.ActivityRequestGet, int64) ([]dto.ActivityResponseGet, int64, error)); ok {
		return rf(ctx, req, userID)
	}

	var r0 []dto.ActivityResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.ActivityResponseGet)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// UnreadCount provides a mock function with given fields: ctx, userID
func (_m *IActivityService) UnreadCount(ctx context.Context, userID int64) (dto.UnreadCountResponse, error) {
	ret := _m.Called(ctx, userID)

	if rf, ok := ret.Get(0).(func(context.Context, int64) (dto.UnreadCountResponse, error)); ok {
		return rf(ctx, userID)
	}

	return ret.Get(0).(dto.UnreadCountResponse), ret.Error(1)
}

// MarkSeen provides a mock function with given fields: ctx, userID
func (_m *IActivityService) MarkSeen(ctx context.Context, userID int64) error {
	ret := _m.Called(ctx, userID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) error); ok {
		r0 = rf(ctx, userID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetApproverNews provides a mock function with given fields: ctx, userID
func (_m *IActivityService) GetApproverNews(ctx context.Context, userID int64) ([]dto.ApproverNewsResponseGet, error) {
	ret := _m.Called(ctx, userID)

	if rf, ok := ret.Get(0).(func(context.Context, int64) ([]dto.ApproverNewsResponseGet, error)); ok {
		return rf(ctx, userID)
	}

	var r0 []dto.ApproverNewsResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.ApproverNewsResponseGet)
	}
	return r0, ret.Error(1)
}

// MarkApproverNewsSeen provides a mock function with given fields: ctx, userID
func (_m *IActivityService) MarkApproverNewsSeen(ctx context.Context, userID int64) error {
	ret := _m.Called(ctx, userID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) error); ok {
		r0 = rf(ctx, userID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewIActivityService creates a new instance of IActivityService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIActivityService(t interface {
	mock.TestingT
	Cleanup(func())
}) *IActivityService {
	m := &IActivityService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
