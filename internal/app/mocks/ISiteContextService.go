// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// ISiteContextService is a mock type for the service.ISiteContextService type
type ISiteContextService struct {
	mock.Mock
}

// GetBySiteID provides a mock function with given fields: ctx, siteID
func (_m *ISiteContextService) GetBySiteID(ctx context.Context, siteID string) (dto.SiteContextResponseGet, error) {
	ret := _m.Called(ctx, siteID)

	if rf, ok := ret.Get(0).(func(context.Context, string) (dto.SiteContextResponseGet, error)); ok {
		return rf(ctx, siteID)
	}

	return ret.Get(0).(dto.SiteContextResponseGet), ret.Error(1)
}

// Update provides a mock function with given fields: ctx, siteID, req, userID
func (_m *ISiteContextService) Update(ctx context.Context, siteID string, req dto.SiteContextRequestUpsert, userID int64) (dto.SiteContextResponseGet, error) {
	ret := _m.Called(ctx, siteID, req, userID)

	if rf, ok := ret.Get(0).(func(context.Context, string, dto.SiteContextRequestUpsert, int64) (dto.SiteContextResponseGet, error)); ok {
		return rf(ctx, siteID, req, userID)
	}

	return ret.Get(0).(dto.SiteContextResponseGet), ret.Error(1)
}

// NewISiteContextService creates a new instance of ISiteContextService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewISiteContextService(t interface {
	mock.TestingT
	Cleanup(func())
}) *ISiteContextService {
	m := &ISiteContextService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
