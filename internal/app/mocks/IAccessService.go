// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// IAccessService is a mock type for the service.IAccessService type
type IAccessService struct {
	mock.Mock
}

// RequireAdmin provides a mock function with given fields: ctx, userID
func (_m *IAccessService) RequireAdmin(ctx context.Context, userID int64) error {
	ret := _m.Called(ctx, userID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) error); ok {
		r0 = rf(ctx, userID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewIAccessService creates a new instance of IAccessService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIAccessService(t interface {
	mock.TestingT
	Cleanup(func())
}) *IAccessService {
	m := &IAccessService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
