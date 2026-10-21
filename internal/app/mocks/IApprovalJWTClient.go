// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// IApprovalJWTClient is a mock type for the client.IApprovalJWTClient type
type IApprovalJWTClient struct {
	mock.Mock
}

// IssueJWT provides a mock function with given fields: ctx, username, nik
func (_m *IApprovalJWTClient) IssueJWT(ctx context.Context, username string, nik string) (string, error) {
	ret := _m.Called(ctx, username, nik)

	if rf, ok := ret.Get(0).(func(context.Context, string, string) (string, error)); ok {
		return rf(ctx, username, nik)
	}

	return ret.Get(0).(string), ret.Error(1)
}

// NewIApprovalJWTClient creates a new instance of IApprovalJWTClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIApprovalJWTClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *IApprovalJWTClient {
	m := &IApprovalJWTClient{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
