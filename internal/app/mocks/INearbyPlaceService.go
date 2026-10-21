// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// INearbyPlaceService is a mock type for the service.INearbyPlaceService type
type INearbyPlaceService struct {
	mock.Mock
}

// GetArea provides a mock function with given fields: ctx, req
func (_m *INearbyPlaceService) GetArea(ctx context.Context, req dto.NearbyPlacesRequestGet) (dto.NearbyAreaResponseGet, error) {
	ret := _m.Called(ctx, req)

	if rf, ok := ret.Get(0).(func(context.Context, dto.NearbyPlacesRequestGet) (dto.NearbyAreaResponseGet, error)); ok {
		return rf(ctx, req)
	}

	return ret.Get(0).(dto.NearbyAreaResponseGet), ret.Error(1)
}

// Measured provides a mock function with given fields: ctx, lat, lng
func (_m *INearbyPlaceService) Measured(ctx context.Context, lat float64, lng float64) (*dto.NearbyMeasured, error) {
	ret := _m.Called(ctx, lat, lng)

	if rf, ok := ret.Get(0).(func(context.Context, float64, float64) (*dto.NearbyMeasured, error)); ok {
		return rf(ctx, lat, lng)
	}

	var r0 *dto.NearbyMeasured
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*dto.NearbyMeasured)
	}
	return r0, ret.Error(1)
}

// Enabled provides a mock function with given fields:
func (_m *INearbyPlaceService) Enabled() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else if ret.Get(0) != nil {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// NewINearbyPlaceService creates a new instance of INearbyPlaceService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewINearbyPlaceService(t interface {
	mock.TestingT
	Cleanup(func())
}) *INearbyPlaceService {
	m := &INearbyPlaceService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
