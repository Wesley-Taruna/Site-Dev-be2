// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// IAILimitRepository is a mock type for the repository.IAILimitRepository type
type IAILimitRepository struct {
	mock.Mock
}

// Take provides a mock function with given fields: ctx, kind, cell, limit
func (_m *IAILimitRepository) Take(ctx context.Context, kind string, cell string, limit int64) (int64, error) {
	ret := _m.Called(ctx, kind, cell, limit)

	if rf, ok := ret.Get(0).(func(context.Context, string, string, int64) (int64, error)); ok {
		return rf(ctx, kind, cell, limit)
	}

	return ret.Get(0).(int64), ret.Error(1)
}

// Release provides a mock function with given fields: ctx, kind, cell
func (_m *IAILimitRepository) Release(ctx context.Context, kind string, cell string) error {
	ret := _m.Called(ctx, kind, cell)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, kind, cell)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Used provides a mock function with given fields: ctx, kind, cell
func (_m *IAILimitRepository) Used(ctx context.Context, kind string, cell string) (int64, error) {
	ret := _m.Called(ctx, kind, cell)

	if rf, ok := ret.Get(0).(func(context.Context, string, string) (int64, error)); ok {
		return rf(ctx, kind, cell)
	}

	return ret.Get(0).(int64), ret.Error(1)
}

// NewIAILimitRepository creates a new instance of IAILimitRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIAILimitRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *IAILimitRepository {
	m := &IAILimitRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
