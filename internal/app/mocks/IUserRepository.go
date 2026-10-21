// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// IUserRepository is a mock type for the repository.IUserRepository type
type IUserRepository struct {
	mock.Mock
}

// GetByID provides a mock function with given fields: ctx, userID
func (_m *IUserRepository) GetByID(ctx context.Context, userID int64) (*model.User, error) {
	ret := _m.Called(ctx, userID)

	if rf, ok := ret.Get(0).(func(context.Context, int64) (*model.User, error)); ok {
		return rf(ctx, userID)
	}

	var r0 *model.User
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.User)
	}
	return r0, ret.Error(1)
}

// NewIUserRepository creates a new instance of IUserRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIUserRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *IUserRepository {
	m := &IUserRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
