// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// IWilayahService is a mock type for the service.IWilayahService type
type IWilayahService struct {
	mock.Mock
}

// SearchByName provides a mock function with given fields: ctx, search, page
func (_m *IWilayahService) SearchByName(ctx context.Context, search string, page int) ([]dto.WilayahSearchResponseGet, int64, error) {
	ret := _m.Called(ctx, search, page)

	if rf, ok := ret.Get(0).(func(context.Context, string, int) ([]dto.WilayahSearchResponseGet, int64, error)); ok {
		return rf(ctx, search, page)
	}

	var r0 []dto.WilayahSearchResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.WilayahSearchResponseGet)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// SearchByPostalCode provides a mock function with given fields: ctx, postalCode, page
func (_m *IWilayahService) SearchByPostalCode(ctx context.Context, postalCode string, page int) ([]dto.WilayahSearchResponseGet, int64, error) {
	ret := _m.Called(ctx, postalCode, page)

	if rf, ok := ret.Get(0).(func(context.Context, string, int) ([]dto.WilayahSearchResponseGet, int64, error)); ok {
		return rf(ctx, postalCode, page)
	}

	var r0 []dto.WilayahSearchResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.WilayahSearchResponseGet)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetProvinces provides a mock function with given fields: ctx
func (_m *IWilayahService) GetProvinces(ctx context.Context) ([]dto.ProvinceResponseGet, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) ([]dto.ProvinceResponseGet, error)); ok {
		return rf(ctx)
	}

	var r0 []dto.ProvinceResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.ProvinceResponseGet)
	}
	return r0, ret.Error(1)
}

// GetProvinceTree provides a mock function with given fields: ctx, provinceCode
func (_m *IWilayahService) GetProvinceTree(ctx context.Context, provinceCode string) (dto.ProvinceTreeResponseGet, error) {
	ret := _m.Called(ctx, provinceCode)

	if rf, ok := ret.Get(0).(func(context.Context, string) (dto.ProvinceTreeResponseGet, error)); ok {
		return rf(ctx, provinceCode)
	}

	return ret.Get(0).(dto.ProvinceTreeResponseGet), ret.Error(1)
}

// NewIWilayahService creates a new instance of IWilayahService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIWilayahService(t interface {
	mock.TestingT
	Cleanup(func())
}) *IWilayahService {
	m := &IWilayahService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
