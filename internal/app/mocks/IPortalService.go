// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// IPortalService is a mock type for the service.IPortalService type
type IPortalService struct {
	mock.Mock
}

// Get provides a mock function with given fields: ctx, req, userID
func (_m *IPortalService) Get(ctx context.Context, req dto.PortalSiteRequestGet, userID int64) ([]dto.PortalSiteResponseGet, int64, error) {
	ret := _m.Called(ctx, req, userID)

	if rf, ok := ret.Get(0).(func(context.Context, dto.PortalSiteRequestGet, int64) ([]dto.PortalSiteResponseGet, int64, error)); ok {
		return rf(ctx, req, userID)
	}

	var r0 []dto.PortalSiteResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.PortalSiteResponseGet)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// CountByGroup provides a mock function with given fields: ctx, userID
func (_m *IPortalService) CountByGroup(ctx context.Context, userID int64) ([]dto.PortalStatusCountResponseGet, error) {
	ret := _m.Called(ctx, userID)

	if rf, ok := ret.Get(0).(func(context.Context, int64) ([]dto.PortalStatusCountResponseGet, error)); ok {
		return rf(ctx, userID)
	}

	var r0 []dto.PortalStatusCountResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.PortalStatusCountResponseGet)
	}
	return r0, ret.Error(1)
}

// GetReport provides a mock function with given fields: ctx, siteID, userID
func (_m *IPortalService) GetReport(ctx context.Context, siteID string, userID int64) (dto.PortalReportResponseGet, error) {
	ret := _m.Called(ctx, siteID, userID)

	if rf, ok := ret.Get(0).(func(context.Context, string, int64) (dto.PortalReportResponseGet, error)); ok {
		return rf(ctx, siteID, userID)
	}

	return ret.Get(0).(dto.PortalReportResponseGet), ret.Error(1)
}

// UpdateAds provides a mock function with given fields: ctx, siteID, req, userID
func (_m *IPortalService) UpdateAds(ctx context.Context, siteID string, req dto.SiteAdsRequestUpdate, userID int64) (dto.SiteAdsResponse, error) {
	ret := _m.Called(ctx, siteID, req, userID)

	if rf, ok := ret.Get(0).(func(context.Context, string, dto.SiteAdsRequestUpdate, int64) (dto.SiteAdsResponse, error)); ok {
		return rf(ctx, siteID, req, userID)
	}

	return ret.Get(0).(dto.SiteAdsResponse), ret.Error(1)
}

// NewIPortalService creates a new instance of IPortalService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIPortalService(t interface {
	mock.TestingT
	Cleanup(func())
}) *IPortalService {
	m := &IPortalService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
