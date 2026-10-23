// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// IAIInsightService is a mock type for the service.IAIInsightService type
type IAIInsightService struct {
	mock.Mock
}

// Generate provides a mock function with given fields: ctx, siteID, actorID
func (_m *IAIInsightService) Generate(ctx context.Context, siteID string, actorID int64) (dto.AIInsightResponseGet, error) {
	ret := _m.Called(ctx, siteID, actorID)

	if rf, ok := ret.Get(0).(func(context.Context, string, int64) (dto.AIInsightResponseGet, error)); ok {
		return rf(ctx, siteID, actorID)
	}

	return ret.Get(0).(dto.AIInsightResponseGet), ret.Error(1)
}

// Get provides a mock function with given fields: ctx, siteID
func (_m *IAIInsightService) Get(ctx context.Context, siteID string) (dto.AIInsightResponseGet, error) {
	ret := _m.Called(ctx, siteID)

	if rf, ok := ret.Get(0).(func(context.Context, string) (dto.AIInsightResponseGet, error)); ok {
		return rf(ctx, siteID)
	}

	return ret.Get(0).(dto.AIInsightResponseGet), ret.Error(1)
}

// Preview provides a mock function with given fields: ctx, req, actorID
func (_m *IAIInsightService) Preview(ctx context.Context, req dto.AIInsightRequestPreview, actorID int64) (dto.AIInsightResponseGet, error) {
	ret := _m.Called(ctx, req, actorID)

	if rf, ok := ret.Get(0).(func(context.Context, dto.AIInsightRequestPreview, int64) (dto.AIInsightResponseGet, error)); ok {
		return rf(ctx, req, actorID)
	}

	return ret.Get(0).(dto.AIInsightResponseGet), ret.Error(1)
}

// Persist provides a mock function with given fields: ctx, siteID, insight, actorID
func (_m *IAIInsightService) Persist(ctx context.Context, siteID string, insight dto.AIInsightResponseGet, actorID int64) error {
	ret := _m.Called(ctx, siteID, insight, actorID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, dto.AIInsightResponseGet, int64) error); ok {
		r0 = rf(ctx, siteID, insight, actorID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetQuota provides a mock function with given fields: ctx, req, actorID
func (_m *IAIInsightService) GetQuota(ctx context.Context, req dto.AIInsightQuotaRequestGet, actorID int64) (dto.AIInsightQuotaResponse, error) {
	ret := _m.Called(ctx, req, actorID)

	if rf, ok := ret.Get(0).(func(context.Context, dto.AIInsightQuotaRequestGet, int64) (dto.AIInsightQuotaResponse, error)); ok {
		return rf(ctx, req, actorID)
	}

	return ret.Get(0).(dto.AIInsightQuotaResponse), ret.Error(1)
}

// FlushQuota provides a mock function with given fields: ctx
func (_m *IAIInsightService) FlushQuota(ctx context.Context) (int, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) (int, error)); ok {
		return rf(ctx)
	}

	return ret.Get(0).(int), ret.Error(1)
}

// NewIAIInsightService creates a new instance of IAIInsightService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIAIInsightService(t interface {
	mock.TestingT
	Cleanup(func())
}) *IAIInsightService {
	m := &IAIInsightService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
