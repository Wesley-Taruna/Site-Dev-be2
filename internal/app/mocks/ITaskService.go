// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// ITaskService is a mock type for the service.ITaskService type
type ITaskService struct {
	mock.Mock
}

// Get provides a mock function with given fields: ctx, req, userID
func (_m *ITaskService) Get(ctx context.Context, req dto.TaskRequestGet, userID int64) ([]dto.TaskResponseGet, int64, error) {
	ret := _m.Called(ctx, req, userID)

	if rf, ok := ret.Get(0).(func(context.Context, dto.TaskRequestGet, int64) ([]dto.TaskResponseGet, int64, error)); ok {
		return rf(ctx, req, userID)
	}

	var r0 []dto.TaskResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.TaskResponseGet)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetHistory provides a mock function with given fields: ctx, siteID
func (_m *ITaskService) GetHistory(ctx context.Context, siteID string) ([]dto.TaskHistoryResponseGet, error) {
	ret := _m.Called(ctx, siteID)

	if rf, ok := ret.Get(0).(func(context.Context, string) ([]dto.TaskHistoryResponseGet, error)); ok {
		return rf(ctx, siteID)
	}

	var r0 []dto.TaskHistoryResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.TaskHistoryResponseGet)
	}
	return r0, ret.Error(1)
}

// GetDetail provides a mock function with given fields: ctx, id, userID
func (_m *ITaskService) GetDetail(ctx context.Context, id string, userID int64) (dto.TaskDetailResponseGet, error) {
	ret := _m.Called(ctx, id, userID)

	if rf, ok := ret.Get(0).(func(context.Context, string, int64) (dto.TaskDetailResponseGet, error)); ok {
		return rf(ctx, id, userID)
	}

	return ret.Get(0).(dto.TaskDetailResponseGet), ret.Error(1)
}

// GetAttachments provides a mock function with given fields: ctx, siteID
func (_m *ITaskService) GetAttachments(ctx context.Context, siteID string) ([]dto.TaskAttachmentResponseGet, error) {
	ret := _m.Called(ctx, siteID)

	if rf, ok := ret.Get(0).(func(context.Context, string) ([]dto.TaskAttachmentResponseGet, error)); ok {
		return rf(ctx, siteID)
	}

	var r0 []dto.TaskAttachmentResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.TaskAttachmentResponseGet)
	}
	return r0, ret.Error(1)
}

// CountByStatus provides a mock function with given fields: ctx, userID
func (_m *ITaskService) CountByStatus(ctx context.Context, userID int64) ([]dto.TaskStatusCountResponseGet, error) {
	ret := _m.Called(ctx, userID)

	if rf, ok := ret.Get(0).(func(context.Context, int64) ([]dto.TaskStatusCountResponseGet, error)); ok {
		return rf(ctx, userID)
	}

	var r0 []dto.TaskStatusCountResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.TaskStatusCountResponseGet)
	}
	return r0, ret.Error(1)
}

// GetSiteList provides a mock function with given fields: ctx, req
func (_m *ITaskService) GetSiteList(ctx context.Context, req dto.ListRequestGet) ([]dto.TaskResponseGet, int64, error) {
	ret := _m.Called(ctx, req)

	if rf, ok := ret.Get(0).(func(context.Context, dto.ListRequestGet) ([]dto.TaskResponseGet, int64, error)); ok {
		return rf(ctx, req)
	}

	var r0 []dto.TaskResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.TaskResponseGet)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetPendingApprovals provides a mock function with given fields: ctx, req, approverID
func (_m *ITaskService) GetPendingApprovals(ctx context.Context, req dto.ListRequestGet, approverID int64) ([]dto.PendingApprovalResponseGet, int64, error) {
	ret := _m.Called(ctx, req, approverID)

	if rf, ok := ret.Get(0).(func(context.Context, dto.ListRequestGet, int64) ([]dto.PendingApprovalResponseGet, int64, error)); ok {
		return rf(ctx, req, approverID)
	}

	var r0 []dto.PendingApprovalResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.PendingApprovalResponseGet)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetApprovalHistory provides a mock function with given fields: ctx, req, approverID
func (_m *ITaskService) GetApprovalHistory(ctx context.Context, req dto.ListRequestGet, approverID int64) ([]dto.ApprovalHistoryResponseGet, int64, error) {
	ret := _m.Called(ctx, req, approverID)

	if rf, ok := ret.Get(0).(func(context.Context, dto.ListRequestGet, int64) ([]dto.ApprovalHistoryResponseGet, int64, error)); ok {
		return rf(ctx, req, approverID)
	}

	var r0 []dto.ApprovalHistoryResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.ApprovalHistoryResponseGet)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetDashboardHistory provides a mock function with given fields: ctx, req
func (_m *ITaskService) GetDashboardHistory(ctx context.Context, req dto.ListRequestGet) ([]dto.DashboardTaskHistoryResponseGet, int64, error) {
	ret := _m.Called(ctx, req)

	if rf, ok := ret.Get(0).(func(context.Context, dto.ListRequestGet) ([]dto.DashboardTaskHistoryResponseGet, int64, error)); ok {
		return rf(ctx, req)
	}

	var r0 []dto.DashboardTaskHistoryResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.DashboardTaskHistoryResponseGet)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetDashboardStatusCount provides a mock function with given fields: ctx
func (_m *ITaskService) GetDashboardStatusCount(ctx context.Context) ([]dto.TaskStatusCountResponseGet, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) ([]dto.TaskStatusCountResponseGet, error)); ok {
		return rf(ctx)
	}

	var r0 []dto.TaskStatusCountResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.TaskStatusCountResponseGet)
	}
	return r0, ret.Error(1)
}

// Approve provides a mock function with given fields: ctx, taskID, actorID, req
func (_m *ITaskService) Approve(ctx context.Context, taskID string, actorID int64, req dto.TaskActionRequest) error {
	ret := _m.Called(ctx, taskID, actorID, req)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, int64, dto.TaskActionRequest) error); ok {
		r0 = rf(ctx, taskID, actorID, req)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewITaskService creates a new instance of ITaskService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewITaskService(t interface {
	mock.TestingT
	Cleanup(func())
}) *ITaskService {
	m := &ITaskService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
