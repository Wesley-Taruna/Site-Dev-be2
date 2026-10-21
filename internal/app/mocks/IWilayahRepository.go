// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// IWilayahRepository is a mock type for the repository.IWilayahRepository type
type IWilayahRepository struct {
	mock.Mock
}

// SearchByName provides a mock function with given fields: ctx, search, limit, offset
func (_m *IWilayahRepository) SearchByName(ctx context.Context, search string, limit int, offset int) ([]model.WilayahSearchRow, int64, error) {
	ret := _m.Called(ctx, search, limit, offset)

	if rf, ok := ret.Get(0).(func(context.Context, string, int, int) ([]model.WilayahSearchRow, int64, error)); ok {
		return rf(ctx, search, limit, offset)
	}

	var r0 []model.WilayahSearchRow
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.WilayahSearchRow)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// SearchByPostalCode provides a mock function with given fields: ctx, postalCode, limit, offset
func (_m *IWilayahRepository) SearchByPostalCode(ctx context.Context, postalCode string, limit int, offset int) ([]model.WilayahSearchRow, int64, error) {
	ret := _m.Called(ctx, postalCode, limit, offset)

	if rf, ok := ret.Get(0).(func(context.Context, string, int, int) ([]model.WilayahSearchRow, int64, error)); ok {
		return rf(ctx, postalCode, limit, offset)
	}

	var r0 []model.WilayahSearchRow
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.WilayahSearchRow)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetProvinces provides a mock function with given fields: ctx
func (_m *IWilayahRepository) GetProvinces(ctx context.Context) ([]model.MProvince, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) ([]model.MProvince, error)); ok {
		return rf(ctx)
	}

	var r0 []model.MProvince
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.MProvince)
	}
	return r0, ret.Error(1)
}

// GetProvinceByCode provides a mock function with given fields: ctx, code
func (_m *IWilayahRepository) GetProvinceByCode(ctx context.Context, code string) (*model.MProvince, error) {
	ret := _m.Called(ctx, code)

	if rf, ok := ret.Get(0).(func(context.Context, string) (*model.MProvince, error)); ok {
		return rf(ctx, code)
	}

	var r0 *model.MProvince
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.MProvince)
	}
	return r0, ret.Error(1)
}

// GetRegenciesByProvince provides a mock function with given fields: ctx, provinceCode
func (_m *IWilayahRepository) GetRegenciesByProvince(ctx context.Context, provinceCode string) ([]model.MRegency, error) {
	ret := _m.Called(ctx, provinceCode)

	if rf, ok := ret.Get(0).(func(context.Context, string) ([]model.MRegency, error)); ok {
		return rf(ctx, provinceCode)
	}

	var r0 []model.MRegency
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.MRegency)
	}
	return r0, ret.Error(1)
}

// GetDistrictsByRegencyCodes provides a mock function with given fields: ctx, regencyCodes
func (_m *IWilayahRepository) GetDistrictsByRegencyCodes(ctx context.Context, regencyCodes []string) ([]model.MDistrict, error) {
	ret := _m.Called(ctx, regencyCodes)

	if rf, ok := ret.Get(0).(func(context.Context, []string) ([]model.MDistrict, error)); ok {
		return rf(ctx, regencyCodes)
	}

	var r0 []model.MDistrict
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.MDistrict)
	}
	return r0, ret.Error(1)
}

// GetVillagesByDistrictCodes provides a mock function with given fields: ctx, districtCodes
func (_m *IWilayahRepository) GetVillagesByDistrictCodes(ctx context.Context, districtCodes []string) ([]model.MVillage, error) {
	ret := _m.Called(ctx, districtCodes)

	if rf, ok := ret.Get(0).(func(context.Context, []string) ([]model.MVillage, error)); ok {
		return rf(ctx, districtCodes)
	}

	var r0 []model.MVillage
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.MVillage)
	}
	return r0, ret.Error(1)
}

// NewIWilayahRepository creates a new instance of IWilayahRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIWilayahRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *IWilayahRepository {
	m := &IWilayahRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
