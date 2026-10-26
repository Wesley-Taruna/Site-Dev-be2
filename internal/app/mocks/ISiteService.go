// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// ISiteService is a mock type for the service.ISiteService type
type ISiteService struct {
	mock.Mock
}

// GetByID provides a mock function with given fields: ctx, siteID
func (_m *ISiteService) GetByID(ctx context.Context, siteID string) (dto.SiteResponseGet, error) {
	ret := _m.Called(ctx, siteID)

	if rf, ok := ret.Get(0).(func(context.Context, string) (dto.SiteResponseGet, error)); ok {
		return rf(ctx, siteID)
	}

	return ret.Get(0).(dto.SiteResponseGet), ret.Error(1)
}

// Get provides a mock function with given fields: ctx, req
func (_m *ISiteService) Get(ctx context.Context, req dto.SiteRequestGet) ([]dto.SiteResponseGet, int64, error) {
	ret := _m.Called(ctx, req)

	if rf, ok := ret.Get(0).(func(context.Context, dto.SiteRequestGet) ([]dto.SiteResponseGet, int64, error)); ok {
		return rf(ctx, req)
	}

	var r0 []dto.SiteResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.SiteResponseGet)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetMaps provides a mock function with given fields: ctx
func (_m *ISiteService) GetMaps(ctx context.Context) (dto.SiteMapListResponseGet, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) (dto.SiteMapListResponseGet, error)); ok {
		return rf(ctx)
	}

	return ret.Get(0).(dto.SiteMapListResponseGet), ret.Error(1)
}

// GetActiveStores provides a mock function with given fields: ctx, req
func (_m *ISiteService) GetActiveStores(ctx context.Context, req dto.ListRequestGet) ([]dto.SiteMapResponseGet, int64, error) {
	ret := _m.Called(ctx, req)

	if rf, ok := ret.Get(0).(func(context.Context, dto.ListRequestGet) ([]dto.SiteMapResponseGet, int64, error)); ok {
		return rf(ctx, req)
	}

	var r0 []dto.SiteMapResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.SiteMapResponseGet)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// UpsertWithStatus provides a mock function with given fields: ctx, req, status, actorID
func (_m *ISiteService) UpsertWithStatus(ctx context.Context, req dto.SiteRequestUpsert, status string, actorID int64) (dto.SiteUpsertResponse, error) {
	ret := _m.Called(ctx, req, status, actorID)

	if rf, ok := ret.Get(0).(func(context.Context, dto.SiteRequestUpsert, string, int64) (dto.SiteUpsertResponse, error)); ok {
		return rf(ctx, req, status, actorID)
	}

	return ret.Get(0).(dto.SiteUpsertResponse), ret.Error(1)
}

// DeleteDraft provides a mock function with given fields: ctx, id, actorID
func (_m *ISiteService) DeleteDraft(ctx context.Context, id string, actorID int64) error {
	ret := _m.Called(ctx, id, actorID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, int64) error); ok {
		r0 = rf(ctx, id, actorID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewISiteService creates a new instance of ISiteService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewISiteService(t interface {
	mock.TestingT
	Cleanup(func())
}) *ISiteService {
	m := &ISiteService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
