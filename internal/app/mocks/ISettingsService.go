// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// ISettingsService is a mock type for the service.ISettingsService type
type ISettingsService struct {
	mock.Mock
}

// GetLimits provides a mock function with given fields: ctx
func (_m *ISettingsService) GetLimits(ctx context.Context) (dto.LimitsResponseGet, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) (dto.LimitsResponseGet, error)); ok {
		return rf(ctx)
	}

	return ret.Get(0).(dto.LimitsResponseGet), ret.Error(1)
}

// UpdateLimits provides a mock function with given fields: ctx, req, updatedBy
func (_m *ISettingsService) UpdateLimits(ctx context.Context, req dto.LimitsRequestUpdate, updatedBy int64) (dto.LimitsResponseGet, error) {
	ret := _m.Called(ctx, req, updatedBy)

	if rf, ok := ret.Get(0).(func(context.Context, dto.LimitsRequestUpdate, int64) (dto.LimitsResponseGet, error)); ok {
		return rf(ctx, req, updatedBy)
	}

	return ret.Get(0).(dto.LimitsResponseGet), ret.Error(1)
}

// FlushUserQuota provides a mock function with given fields: ctx, req
func (_m *ISettingsService) FlushUserQuota(ctx context.Context, req dto.QuotaFlushRequest) (dto.QuotaFlushResponse, error) {
	ret := _m.Called(ctx, req)

	if rf, ok := ret.Get(0).(func(context.Context, dto.QuotaFlushRequest) (dto.QuotaFlushResponse, error)); ok {
		return rf(ctx, req)
	}

	return ret.Get(0).(dto.QuotaFlushResponse), ret.Error(1)
}

// NewISettingsService creates a new instance of ISettingsService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewISettingsService(t interface {
	mock.TestingT
	Cleanup(func())
}) *ISettingsService {
	m := &ISettingsService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
