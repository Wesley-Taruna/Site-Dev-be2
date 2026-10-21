// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// IUserService is a mock type for the service.IUserService type
type IUserService struct {
	mock.Mock
}

// Get provides a mock function with given fields: ctx
func (_m *IUserService) Get(ctx context.Context) ([]dto.UserResponseGet, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) ([]dto.UserResponseGet, error)); ok {
		return rf(ctx)
	}

	var r0 []dto.UserResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.UserResponseGet)
	}
	return r0, ret.Error(1)
}

// Create provides a mock function with given fields: ctx, req
func (_m *IUserService) Create(ctx context.Context, req dto.UserRequestCreate) (dto.UserResponseGet, error) {
	ret := _m.Called(ctx, req)

	if rf, ok := ret.Get(0).(func(context.Context, dto.UserRequestCreate) (dto.UserResponseGet, error)); ok {
		return rf(ctx, req)
	}

	return ret.Get(0).(dto.UserResponseGet), ret.Error(1)
}

// Update provides a mock function with given fields: ctx, userID, req
func (_m *IUserService) Update(ctx context.Context, userID int64, req dto.UserRequestUpdate) (dto.UserResponseGet, error) {
	ret := _m.Called(ctx, userID, req)

	if rf, ok := ret.Get(0).(func(context.Context, int64, dto.UserRequestUpdate) (dto.UserResponseGet, error)); ok {
		return rf(ctx, userID, req)
	}

	return ret.Get(0).(dto.UserResponseGet), ret.Error(1)
}

// Delete provides a mock function with given fields: ctx, userID
func (_m *IUserService) Delete(ctx context.Context, userID int64) error {
	ret := _m.Called(ctx, userID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) error); ok {
		r0 = rf(ctx, userID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewIUserService creates a new instance of IUserService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIUserService(t interface {
	mock.TestingT
	Cleanup(func())
}) *IUserService {
	m := &IUserService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
