// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// IQuotaRepository is a mock type for the repository.IQuotaRepository type
type IQuotaRepository struct {
	mock.Mock
}

// Limit provides a mock function with given fields: ctx, kind
func (_m *IQuotaRepository) Limit(ctx context.Context, kind string) (int64, error) {
	ret := _m.Called(ctx, kind)

	if rf, ok := ret.Get(0).(func(context.Context, string) (int64, error)); ok {
		return rf(ctx, kind)
	}

	return ret.Get(0).(int64), ret.Error(1)
}

// Remaining provides a mock function with given fields: ctx, kind, actorID
func (_m *IQuotaRepository) Remaining(ctx context.Context, kind string, actorID int64) (int64, error) {
	ret := _m.Called(ctx, kind, actorID)

	if rf, ok := ret.Get(0).(func(context.Context, string, int64) (int64, error)); ok {
		return rf(ctx, kind, actorID)
	}

	return ret.Get(0).(int64), ret.Error(1)
}

// Consume provides a mock function with given fields: ctx, kind, actorID
func (_m *IQuotaRepository) Consume(ctx context.Context, kind string, actorID int64) (int64, error) {
	ret := _m.Called(ctx, kind, actorID)

	if rf, ok := ret.Get(0).(func(context.Context, string, int64) (int64, error)); ok {
		return rf(ctx, kind, actorID)
	}

	return ret.Get(0).(int64), ret.Error(1)
}

// FlushUser provides a mock function with given fields: ctx, kind, actorID
func (_m *IQuotaRepository) FlushUser(ctx context.Context, kind string, actorID int64) error {
	ret := _m.Called(ctx, kind, actorID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, int64) error); ok {
		r0 = rf(ctx, kind, actorID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// FlushAll provides a mock function with given fields: ctx, kind
func (_m *IQuotaRepository) FlushAll(ctx context.Context, kind string) (int, error) {
	ret := _m.Called(ctx, kind)

	if rf, ok := ret.Get(0).(func(context.Context, string) (int, error)); ok {
		return rf(ctx, kind)
	}

	return ret.Get(0).(int), ret.Error(1)
}

// NewIQuotaRepository creates a new instance of IQuotaRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIQuotaRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *IQuotaRepository {
	m := &IQuotaRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
