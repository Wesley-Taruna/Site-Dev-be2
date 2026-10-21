// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// ITaskRepository is a mock type for the repository.ITaskRepository type
type ITaskRepository struct {
	mock.Mock
}

// Get provides a mock function with given fields: ctx, offset, limit, orderBy, siteIDs, status, createdBy
func (_m *ITaskRepository) Get(ctx context.Context, offset int, limit int, orderBy string, siteIDs []string, status string, createdBy int64) ([]*model.Task, int64, error) {
	ret := _m.Called(ctx, offset, limit, orderBy, siteIDs, status, createdBy)

	if rf, ok := ret.Get(0).(func(context.Context, int, int, string, []string, string, int64) ([]*model.Task, int64, error)); ok {
		return rf(ctx, offset, limit, orderBy, siteIDs, status, createdBy)
	}

	var r0 []*model.Task
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.Task)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetBySiteID provides a mock function with given fields: ctx, siteID
func (_m *ITaskRepository) GetBySiteID(ctx context.Context, siteID string) (*model.Task, error) {
	ret := _m.Called(ctx, siteID)

	if rf, ok := ret.Get(0).(func(context.Context, string) (*model.Task, error)); ok {
		return rf(ctx, siteID)
	}

	var r0 *model.Task
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.Task)
	}
	return r0, ret.Error(1)
}

// GetByID provides a mock function with given fields: ctx, taskID
func (_m *ITaskRepository) GetByID(ctx context.Context, taskID string) (*model.Task, error) {
	ret := _m.Called(ctx, taskID)

	if rf, ok := ret.Get(0).(func(context.Context, string) (*model.Task, error)); ok {
		return rf(ctx, taskID)
	}

	var r0 *model.Task
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.Task)
	}
	return r0, ret.Error(1)
}

// GetByIDs provides a mock function with given fields: ctx, taskIDs
func (_m *ITaskRepository) GetByIDs(ctx context.Context, taskIDs []string) ([]*model.Task, error) {
	ret := _m.Called(ctx, taskIDs)

	if rf, ok := ret.Get(0).(func(context.Context, []string) ([]*model.Task, error)); ok {
		return rf(ctx, taskIDs)
	}

	var r0 []*model.Task
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.Task)
	}
	return r0, ret.Error(1)
}

// GetByStatuses provides a mock function with given fields: ctx, offset, limit, orderBy, statuses, siteIDs
func (_m *ITaskRepository) GetByStatuses(ctx context.Context, offset int, limit int, orderBy string, statuses []string, siteIDs []string) ([]*model.Task, int64, error) {
	ret := _m.Called(ctx, offset, limit, orderBy, statuses, siteIDs)

	if rf, ok := ret.Get(0).(func(context.Context, int, int, string, []string, []string) ([]*model.Task, int64, error)); ok {
		return rf(ctx, offset, limit, orderBy, statuses, siteIDs)
	}

	var r0 []*model.Task
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.Task)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// CountByStatus provides a mock function with given fields: ctx, createdBy
func (_m *ITaskRepository) CountByStatus(ctx context.Context, createdBy int64) ([]model.TaskStatusCount, error) {
	ret := _m.Called(ctx, createdBy)

	if rf, ok := ret.Get(0).(func(context.Context, int64) ([]model.TaskStatusCount, error)); ok {
		return rf(ctx, createdBy)
	}

	var r0 []model.TaskStatusCount
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.TaskStatusCount)
	}
	return r0, ret.Error(1)
}

// NewITaskRepository creates a new instance of ITaskRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewITaskRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *ITaskRepository {
	m := &ITaskRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
