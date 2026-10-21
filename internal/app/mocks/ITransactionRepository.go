// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// ITransactionRepository is a mock type for the repository.ITransactionRepository type
type ITransactionRepository struct {
	mock.Mock
}

// Do provides a mock function with given fields: ctx, fn
func (_m *ITransactionRepository) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	ret := _m.Called(ctx, fn)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, func(ctx context.Context) error) error); ok {
		r0 = rf(ctx, fn)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewITransactionRepository creates a new instance of ITransactionRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewITransactionRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *ITransactionRepository {
	m := &ITransactionRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
