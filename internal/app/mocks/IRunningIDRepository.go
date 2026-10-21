// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// IRunningIDRepository is a mock type for the repository.IRunningIDRepository type
type IRunningIDRepository struct {
	mock.Mock
}

// NextSiteID provides a mock function with given fields: ctx
func (_m *IRunningIDRepository) NextSiteID(ctx context.Context) (string, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) (string, error)); ok {
		return rf(ctx)
	}

	return ret.Get(0).(string), ret.Error(1)
}

// NextTaskID provides a mock function with given fields: ctx
func (_m *IRunningIDRepository) NextTaskID(ctx context.Context) (string, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) (string, error)); ok {
		return rf(ctx)
	}

	return ret.Get(0).(string), ret.Error(1)
}

// NewIRunningIDRepository creates a new instance of IRunningIDRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIRunningIDRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *IRunningIDRepository {
	m := &IRunningIDRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
