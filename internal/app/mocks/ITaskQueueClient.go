// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// ITaskQueueClient is a mock type for the client.ITaskQueueClient type
type ITaskQueueClient struct {
	mock.Mock
}

// Enabled provides a mock function with given fields:
func (_m *ITaskQueueClient) Enabled() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else if ret.Get(0) != nil {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Enqueue provides a mock function with given fields: ctx, name, path, body
func (_m *ITaskQueueClient) Enqueue(ctx context.Context, name string, path string, body []byte) error {
	ret := _m.Called(ctx, name, path, body)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, []byte) error); ok {
		r0 = rf(ctx, name, path, body)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewITaskQueueClient creates a new instance of ITaskQueueClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewITaskQueueClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *ITaskQueueClient {
	m := &ITaskQueueClient{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
