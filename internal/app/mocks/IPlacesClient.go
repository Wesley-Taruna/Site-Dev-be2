// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/client"

	"github.com/stretchr/testify/mock"
)

// IPlacesClient is a mock type for the client.IPlacesClient type
type IPlacesClient struct {
	mock.Mock
}

// Configured provides a mock function with given fields:
func (_m *IPlacesClient) Configured() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else if ret.Get(0) != nil {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// FetchArea provides a mock function with given fields: ctx, lat, lng, radiusM, takeCall
func (_m *IPlacesClient) FetchArea(ctx context.Context, lat float64, lng float64, radiusM int, takeCall func(ctx context.Context) error) (client.
	PlacesArea, error) {
	ret := _m.Called(ctx, lat, lng, radiusM, takeCall)

	if rf, ok := ret.Get(0).(func(context.Context, float64, float64, int, func(ctx context.Context) error) (client.
		PlacesArea, error)); ok {
		return rf(ctx, lat, lng, radiusM, takeCall)
	}

	return ret.Get(0).(client.
		PlacesArea), ret.Error(1)
}

// NewIPlacesClient creates a new instance of IPlacesClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIPlacesClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *IPlacesClient {
	m := &IPlacesClient{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
