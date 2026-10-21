// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// IQueueService is a mock type for the service.IQueueService type
type IQueueService struct {
	mock.Mock
}

// SendApprovalEmail provides a mock function with given fields: ctx, payload
func (_m *IQueueService) SendApprovalEmail(ctx context.Context, payload dto.ApprovalEmailTaskPayload) error {
	ret := _m.Called(ctx, payload)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, dto.ApprovalEmailTaskPayload) error); ok {
		r0 = rf(ctx, payload)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewIQueueService creates a new instance of IQueueService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIQueueService(t interface {
	mock.TestingT
	Cleanup(func())
}) *IQueueService {
	m := &IQueueService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
