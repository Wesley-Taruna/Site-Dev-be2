// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// ISystemSettingRepository is a mock type for the repository.ISystemSettingRepository type
type ISystemSettingRepository struct {
	mock.Mock
}

// GetByKey provides a mock function with given fields: ctx, key
func (_m *ISystemSettingRepository) GetByKey(ctx context.Context, key string) (*model.SystemSetting, error) {
	ret := _m.Called(ctx, key)

	if rf, ok := ret.Get(0).(func(context.Context, string) (*model.SystemSetting, error)); ok {
		return rf(ctx, key)
	}

	var r0 *model.SystemSetting
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.SystemSetting)
	}
	return r0, ret.Error(1)
}

// GetInt provides a mock function with given fields: ctx, key, fallback
func (_m *ISystemSettingRepository) GetInt(ctx context.Context, key string, fallback int64) (int64, error) {
	ret := _m.Called(ctx, key, fallback)

	if rf, ok := ret.Get(0).(func(context.Context, string, int64) (int64, error)); ok {
		return rf(ctx, key, fallback)
	}

	return ret.Get(0).(int64), ret.Error(1)
}

// UpsertInt provides a mock function with given fields: ctx, key, value, updatedBy
func (_m *ISystemSettingRepository) UpsertInt(ctx context.Context, key string, value int64, updatedBy int64) error {
	ret := _m.Called(ctx, key, value, updatedBy)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, int64, int64) error); ok {
		r0 = rf(ctx, key, value, updatedBy)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewISystemSettingRepository creates a new instance of ISystemSettingRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewISystemSettingRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *ISystemSettingRepository {
	m := &ISystemSettingRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
