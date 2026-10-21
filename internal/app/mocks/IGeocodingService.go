// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// IGeocodingService is a mock type for the service.IGeocodingService type
type IGeocodingService struct {
	mock.Mock
}

// Reverse provides a mock function with given fields: ctx, lat, lng, actorID
func (_m *IGeocodingService) Reverse(ctx context.Context, lat float64, lng float64, actorID int64) (dto.ReverseGeocodeResponse, error) {
	ret := _m.Called(ctx, lat, lng, actorID)

	if rf, ok := ret.Get(0).(func(context.Context, float64, float64, int64) (dto.ReverseGeocodeResponse, error)); ok {
		return rf(ctx, lat, lng, actorID)
	}

	return ret.Get(0).(dto.ReverseGeocodeResponse), ret.Error(1)
}

// Search provides a mock function with given fields: ctx, query, actorID
func (_m *IGeocodingService) Search(ctx context.Context, query string, actorID int64) ([]dto.AddressSearchResult, error) {
	ret := _m.Called(ctx, query, actorID)

	if rf, ok := ret.Get(0).(func(context.Context, string, int64) ([]dto.AddressSearchResult, error)); ok {
		return rf(ctx, query, actorID)
	}

	var r0 []dto.AddressSearchResult
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.AddressSearchResult)
	}
	return r0, ret.Error(1)
}

// NewIGeocodingService creates a new instance of IGeocodingService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIGeocodingService(t interface {
	mock.TestingT
	Cleanup(func())
}) *IGeocodingService {
	m := &IGeocodingService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
