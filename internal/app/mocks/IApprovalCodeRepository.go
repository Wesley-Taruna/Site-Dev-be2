// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// IApprovalCodeRepository is a mock type for the repository.IApprovalCodeRepository type
type IApprovalCodeRepository struct {
	mock.Mock
}

// Set provides a mock function with given fields: ctx, code, detailID
func (_m *IApprovalCodeRepository) Set(ctx context.Context, code string, detailID uint) error {
	ret := _m.Called(ctx, code, detailID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, uint) error); ok {
		r0 = rf(ctx, code, detailID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Get provides a mock function with given fields: ctx, code
func (_m *IApprovalCodeRepository) Get(ctx context.Context, code string) (uint, error) {
	ret := _m.Called(ctx, code)

	if rf, ok := ret.Get(0).(func(context.Context, string) (uint, error)); ok {
		return rf(ctx, code)
	}

	return ret.Get(0).(uint), ret.Error(1)
}

// NewIApprovalCodeRepository creates a new instance of IApprovalCodeRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIApprovalCodeRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *IApprovalCodeRepository {
	m := &IApprovalCodeRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
