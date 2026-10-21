// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/client"

	"github.com/stretchr/testify/mock"
)

// IMailClient is a mock type for the client.IMailClient type
type IMailClient struct {
	mock.Mock
}

// SendApprovalPending provides a mock function with given fields: ctx, to, toName, data
func (_m *IMailClient) SendApprovalPending(ctx context.Context, to string, toName string, data client.
	ApprovalPendingEmail) error {
	ret := _m.Called(ctx, to, toName, data)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, client.
		ApprovalPendingEmail) error); ok {
		r0 = rf(ctx, to, toName, data)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewIMailClient creates a new instance of IMailClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIMailClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *IMailClient {
	m := &IMailClient{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
