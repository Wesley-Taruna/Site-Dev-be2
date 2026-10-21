// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// INearbyAreaRepository is a mock type for the repository.INearbyAreaRepository type
type INearbyAreaRepository struct {
	mock.Mock
}

// GetArea provides a mock function with given fields: ctx, key
func (_m *INearbyAreaRepository) GetArea(ctx context.Context, key string) ([]byte, bool) {
	ret := _m.Called(ctx, key)

	if rf, ok := ret.Get(0).(func(context.Context, string) ([]byte, bool)); ok {
		return rf(ctx, key)
	}

	var r0 []byte
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]byte)
	}
	return r0, ret.Get(1).(bool)
}

// SetArea provides a mock function with given fields: ctx, key, raw, ttl
func (_m *INearbyAreaRepository) SetArea(ctx context.Context, key string, raw []byte, ttl time.Duration) {
	_m.Called(ctx, key, raw, ttl)
}

// TakeGoogleCall provides a mock function with given fields: ctx, dailyCap
func (_m *INearbyAreaRepository) TakeGoogleCall(ctx context.Context, dailyCap int64) error {
	ret := _m.Called(ctx, dailyCap)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) error); ok {
		r0 = rf(ctx, dailyCap)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GoogleCallsLeft provides a mock function with given fields: ctx, dailyCap
func (_m *INearbyAreaRepository) GoogleCallsLeft(ctx context.Context, dailyCap int64) int64 {
	ret := _m.Called(ctx, dailyCap)

	var r0 int64
	if rf, ok := ret.Get(0).(func(context.Context, int64) int64); ok {
		r0 = rf(ctx, dailyCap)
	} else if ret.Get(0) != nil {
		r0 = ret.Get(0).(int64)
	}

	return r0
}

// NewINearbyAreaRepository creates a new instance of INearbyAreaRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewINearbyAreaRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *INearbyAreaRepository {
	m := &INearbyAreaRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
