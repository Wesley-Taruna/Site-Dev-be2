// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/client"
	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// IOverpassClient is a mock type for the client.IOverpassClient type
type IOverpassClient struct {
	mock.Mock
}

// FetchArea provides a mock function with given fields: ctx, lat, lng, radiusM, brands
func (_m *IOverpassClient) FetchArea(ctx context.Context, lat float64, lng float64, radiusM int, brands client.BrandSet) ([]dto.NearbyPlaceItem, error) {
	ret := _m.Called(ctx, lat, lng, radiusM, brands)

	if rf, ok := ret.Get(0).(func(context.Context, float64, float64, int, client.BrandSet) ([]dto.NearbyPlaceItem, error)); ok {
		return rf(ctx, lat, lng, radiusM, brands)
	}

	var r0 []dto.NearbyPlaceItem
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.NearbyPlaceItem)
	}
	return r0, ret.Error(1)
}

// NewIOverpassClient creates a new instance of IOverpassClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIOverpassClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *IOverpassClient {
	m := &IOverpassClient{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
