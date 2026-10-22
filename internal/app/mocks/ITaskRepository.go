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

// GetByStatuses provides a mock function with given fields: ctx, offset, limit, orderBy, statuses, siteIDs, createdBy
func (_m *ITaskRepository) GetByStatuses(ctx context.Context, offset int, limit int, orderBy string, statuses []string, siteIDs []string, createdBy int64) ([]*model.Task, int64, error) {
	ret := _m.Called(ctx, offset, limit, orderBy, statuses, siteIDs, createdBy)

	if rf, ok := ret.Get(0).(func(context.Context, int, int, string, []string, []string, int64) ([]*model.Task, int64, error)); ok {
		return rf(ctx, offset, limit, orderBy, statuses, siteIDs, createdBy)
	}

	var r0 []*model.Task
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.Task)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetNotInStatuses provides a mock function with given fields: ctx, offset, limit, orderBy, excluded, siteIDs, createdBy
func (_m *ITaskRepository) GetNotInStatuses(ctx context.Context, offset int, limit int, orderBy string, excluded []string, siteIDs []string, createdBy int64) ([]*model.Task, int64, error) {
	ret := _m.Called(ctx, offset, limit, orderBy, excluded, siteIDs, createdBy)

	if rf, ok := ret.Get(0).(func(context.Context, int, int, string, []string, []string, int64) ([]*model.Task, int64, error)); ok {
		return rf(ctx, offset, limit, orderBy, excluded, siteIDs, createdBy)
	}

	var r0 []*model.Task
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.Task)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetStatusesBySiteIDs provides a mock function with given fields: ctx, siteIDs
func (_m *ITaskRepository) GetStatusesBySiteIDs(ctx context.Context, siteIDs []string) ([]*model.Task, error) {
	ret := _m.Called(ctx, siteIDs)

	if rf, ok := ret.Get(0).(func(context.Context, []string) ([]*model.Task, error)); ok {
		return rf(ctx, siteIDs)
	}

	var r0 []*model.Task
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.Task)
	}
	return r0, ret.Error(1)
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

// Create provides a mock function with given fields: ctx, task
func (_m *ITaskRepository) Create(ctx context.Context, task *model.Task) error {
	ret := _m.Called(ctx, task)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *model.Task) error); ok {
		r0 = rf(ctx, task)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateStatus provides a mock function with given fields: ctx, taskID, updates
func (_m *ITaskRepository) UpdateStatus(ctx context.Context, taskID string, updates map[string]interface{}) error {
	ret := _m.Called(ctx, taskID, updates)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, map[string]interface{}) error); ok {
		r0 = rf(ctx, taskID, updates)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetActiveRequestPoints provides a mock function with given fields: ctx, minLat, maxLat, minLon, maxLon, extraStatuses
func (_m *ITaskRepository) GetActiveRequestPoints(ctx context.Context, minLat float64, maxLat float64, minLon float64, maxLon float64, extraStatuses []string) ([]model.RequestPoint, error) {
	ret := _m.Called(ctx, minLat, maxLat, minLon, maxLon, extraStatuses)

	if rf, ok := ret.Get(0).(func(context.Context, float64, float64, float64, float64, []string) ([]model.RequestPoint, error)); ok {
		return rf(ctx, minLat, maxLat, minLon, maxLon, extraStatuses)
	}

	var r0 []model.RequestPoint
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.RequestPoint)
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
