// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"
	"time"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// ITaskHistoryRepository is a mock type for the repository.ITaskHistoryRepository type
type ITaskHistoryRepository struct {
	mock.Mock
}

// GetByTaskID provides a mock function with given fields: ctx, taskID
func (_m *ITaskHistoryRepository) GetByTaskID(ctx context.Context, taskID string) ([]*model.TaskHistory, error) {
	ret := _m.Called(ctx, taskID)

	if rf, ok := ret.Get(0).(func(context.Context, string) ([]*model.TaskHistory, error)); ok {
		return rf(ctx, taskID)
	}

	var r0 []*model.TaskHistory
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.TaskHistory)
	}
	return r0, ret.Error(1)
}

// Get provides a mock function with given fields: ctx, offset, limit, orderBy
func (_m *ITaskHistoryRepository) Get(ctx context.Context, offset int, limit int, orderBy string) ([]*model.TaskHistory, int64, error) {
	ret := _m.Called(ctx, offset, limit, orderBy)

	if rf, ok := ret.Get(0).(func(context.Context, int, int, string) ([]*model.TaskHistory, int64, error)); ok {
		return rf(ctx, offset, limit, orderBy)
	}

	var r0 []*model.TaskHistory
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.TaskHistory)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetFeedByCreator provides a mock function with given fields: ctx, creatorID, actions, notActorID, taskStatus, offset, limit
func (_m *ITaskHistoryRepository) GetFeedByCreator(ctx context.Context, creatorID int64, actions []string, notActorID int64, taskStatus string, offset int, limit int) ([]*model.TaskHistory, int64, error) {
	ret := _m.Called(ctx, creatorID, actions, notActorID, taskStatus, offset, limit)

	if rf, ok := ret.Get(0).(func(context.Context, int64, []string, int64, string, int, int) ([]*model.TaskHistory, int64, error)); ok {
		return rf(ctx, creatorID, actions, notActorID, taskStatus, offset, limit)
	}

	var r0 []*model.TaskHistory
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.TaskHistory)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// CountFeedSince provides a mock function with given fields: ctx, creatorID, actions, since
func (_m *ITaskHistoryRepository) CountFeedSince(ctx context.Context, creatorID int64, actions []string, since time.Time) (int64, error) {
	ret := _m.Called(ctx, creatorID, actions, since)

	if rf, ok := ret.Get(0).(func(context.Context, int64, []string, time.Time) (int64, error)); ok {
		return rf(ctx, creatorID, actions, since)
	}

	return ret.Get(0).(int64), ret.Error(1)
}

// GetFeedByApprover provides a mock function with given fields: ctx, approverID, actions, actorID, waitingOnly, offset, limit
func (_m *ITaskHistoryRepository) GetFeedByApprover(ctx context.Context, approverID int64, actions []string, actorID int64, waitingOnly bool, offset int, limit int) ([]*model.TaskHistory, int64, error) {
	ret := _m.Called(ctx, approverID, actions, actorID, waitingOnly, offset, limit)

	if rf, ok := ret.Get(0).(func(context.Context, int64, []string, int64, bool, int, int) ([]*model.TaskHistory, int64, error)); ok {
		return rf(ctx, approverID, actions, actorID, waitingOnly, offset, limit)
	}

	var r0 []*model.TaskHistory
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.TaskHistory)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// CountFeedByApproverSince provides a mock function with given fields: ctx, approverID, actions, since
func (_m *ITaskHistoryRepository) CountFeedByApproverSince(ctx context.Context, approverID int64, actions []string, since time.Time) (int64, error) {
	ret := _m.Called(ctx, approverID, actions, since)

	if rf, ok := ret.Get(0).(func(context.Context, int64, []string, time.Time) (int64, error)); ok {
		return rf(ctx, approverID, actions, since)
	}

	return ret.Get(0).(int64), ret.Error(1)
}

// GetTaskIDsWaitingFor provides a mock function with given fields: ctx, approverID, taskIDs
func (_m *ITaskHistoryRepository) GetTaskIDsWaitingFor(ctx context.Context, approverID int64, taskIDs []string) ([]string, error) {
	ret := _m.Called(ctx, approverID, taskIDs)

	if rf, ok := ret.Get(0).(func(context.Context, int64, []string) ([]string, error)); ok {
		return rf(ctx, approverID, taskIDs)
	}

	var r0 []string
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]string)
	}
	return r0, ret.Error(1)
}

// GetByTaskIDsAndActor provides a mock function with given fields: ctx, taskIDs, actorID, action
func (_m *ITaskHistoryRepository) GetByTaskIDsAndActor(ctx context.Context, taskIDs []string, actorID int64, action string) ([]*model.TaskHistory, error) {
	ret := _m.Called(ctx, taskIDs, actorID, action)

	if rf, ok := ret.Get(0).(func(context.Context, []string, int64, string) ([]*model.TaskHistory, error)); ok {
		return rf(ctx, taskIDs, actorID, action)
	}

	var r0 []*model.TaskHistory
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.TaskHistory)
	}
	return r0, ret.Error(1)
}

// Create provides a mock function with given fields: ctx, history
func (_m *ITaskHistoryRepository) Create(ctx context.Context, history *model.TaskHistory) error {
	ret := _m.Called(ctx, history)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *model.TaskHistory) error); ok {
		r0 = rf(ctx, history)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewITaskHistoryRepository creates a new instance of ITaskHistoryRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewITaskHistoryRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *ITaskHistoryRepository {
	m := &ITaskHistoryRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
