// Written by hand in the layout mockery (v2) generates, because mockery is
// not installed on this machine. Regenerate with mockery once it is
// (see README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

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

	var r0 []*model.TaskHistory
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.TaskHistory)
	}
	return r0, ret.Error(1)
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
