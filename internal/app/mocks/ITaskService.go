// Written by hand in the layout mockery (v2) generates, because mockery is
// not installed on this machine. Regenerate with mockery once it is
// (see README "Testing"); the type and constructor names stay the same.

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

	var r0 []dto.TaskResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.TaskResponseGet)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetHistory provides a mock function with given fields: ctx, siteID
func (_m *ITaskService) GetHistory(ctx context.Context, siteID string) ([]dto.TaskHistoryResponseGet, error) {
	ret := _m.Called(ctx, siteID)

	var r0 []dto.TaskHistoryResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.TaskHistoryResponseGet)
	}
	return r0, ret.Error(1)
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
