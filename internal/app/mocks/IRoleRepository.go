// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// IRoleRepository is a mock type for the repository.IRoleRepository type
type IRoleRepository struct {
	mock.Mock
}

// GetByID provides a mock function with given fields: ctx, roleID
func (_m *IRoleRepository) GetByID(ctx context.Context, roleID string) (*model.Role, error) {
	ret := _m.Called(ctx, roleID)

	if rf, ok := ret.Get(0).(func(context.Context, string) (*model.Role, error)); ok {
		return rf(ctx, roleID)
	}

	var r0 *model.Role
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.Role)
	}
	return r0, ret.Error(1)
}

// Get provides a mock function with given fields: ctx
func (_m *IRoleRepository) Get(ctx context.Context) ([]*model.Role, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) ([]*model.Role, error)); ok {
		return rf(ctx)
	}

	var r0 []*model.Role
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.Role)
	}
	return r0, ret.Error(1)
}

// NewIRoleRepository creates a new instance of IRoleRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIRoleRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *IRoleRepository {
	m := &IRoleRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
