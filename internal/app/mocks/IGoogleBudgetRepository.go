// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// IGoogleBudgetRepository is a mock type for the repository.IGoogleBudgetRepository type
type IGoogleBudgetRepository struct {
	mock.Mock
}

// Take provides a mock function with given fields: ctx, api, dailyCap
func (_m *IGoogleBudgetRepository) Take(ctx context.Context, api string, dailyCap int64) error {
	ret := _m.Called(ctx, api, dailyCap)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, int64) error); ok {
		r0 = rf(ctx, api, dailyCap)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewIGoogleBudgetRepository creates a new instance of IGoogleBudgetRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIGoogleBudgetRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *IGoogleBudgetRepository {
	m := &IGoogleBudgetRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
