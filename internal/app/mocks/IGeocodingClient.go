// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// IGeocodingClient is a mock type for the client.IGeocodingClient type
type IGeocodingClient struct {
	mock.Mock
}

// Configured provides a mock function with given fields:
func (_m *IGeocodingClient) Configured() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else if ret.Get(0) != nil {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Reverse provides a mock function with given fields: ctx, lat, lng
func (_m *IGeocodingClient) Reverse(ctx context.Context, lat float64, lng float64) (dto.ReverseGeocodeResponse, error) {
	ret := _m.Called(ctx, lat, lng)

	if rf, ok := ret.Get(0).(func(context.Context, float64, float64) (dto.ReverseGeocodeResponse, error)); ok {
		return rf(ctx, lat, lng)
	}

	return ret.Get(0).(dto.ReverseGeocodeResponse), ret.Error(1)
}

// Search provides a mock function with given fields: ctx, query
func (_m *IGeocodingClient) Search(ctx context.Context, query string) ([]dto.AddressSearchResult, error) {
	ret := _m.Called(ctx, query)

	if rf, ok := ret.Get(0).(func(context.Context, string) ([]dto.AddressSearchResult, error)); ok {
		return rf(ctx, query)
	}

	var r0 []dto.AddressSearchResult
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.AddressSearchResult)
	}
	return r0, ret.Error(1)
}

// NewIGeocodingClient creates a new instance of IGeocodingClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIGeocodingClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *IGeocodingClient {
	m := &IGeocodingClient{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
