// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// IFamilyMartStoreRepository is a mock type for the repository.IFamilyMartStoreRepository type
type IFamilyMartStoreRepository struct {
	mock.Mock
}

// InBox provides a mock function with given fields: ctx, minLat, maxLat, minLng, maxLng, limit
func (_m *IFamilyMartStoreRepository) InBox(ctx context.Context, minLat float64, maxLat float64, minLng float64, maxLng float64, limit int) ([]model.FamilyMartStore, error) {
	ret := _m.Called(ctx, minLat, maxLat, minLng, maxLng, limit)

	if rf, ok := ret.Get(0).(func(context.Context, float64, float64, float64, float64, int) ([]model.FamilyMartStore, error)); ok {
		return rf(ctx, minLat, maxLat, minLng, maxLng, limit)
	}

	var r0 []model.FamilyMartStore
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.FamilyMartStore)
	}
	return r0, ret.Error(1)
}

// CacheTag provides a mock function with given fields:
func (_m *IFamilyMartStoreRepository) CacheTag() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// NewIFamilyMartStoreRepository creates a new instance of IFamilyMartStoreRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIFamilyMartStoreRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *IFamilyMartStoreRepository {
	m := &IFamilyMartStoreRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
