// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// IRoleService is a mock type for the service.IRoleService type
type IRoleService struct {
	mock.Mock
}

// Get provides a mock function with given fields: ctx
func (_m *IRoleService) Get(ctx context.Context) ([]dto.RoleResponseGet, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) ([]dto.RoleResponseGet, error)); ok {
		return rf(ctx)
	}

	var r0 []dto.RoleResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.RoleResponseGet)
	}
	return r0, ret.Error(1)
}

// NewIRoleService creates a new instance of IRoleService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIRoleService(t interface {
	mock.TestingT
	Cleanup(func())
}) *IRoleService {
	m := &IRoleService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
