// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// ISiteAdsRepository is a mock type for the repository.ISiteAdsRepository type
type ISiteAdsRepository struct {
	mock.Mock
}

// GetBySiteIDs provides a mock function with given fields: ctx, siteIDs
func (_m *ISiteAdsRepository) GetBySiteIDs(ctx context.Context, siteIDs []string) ([]*model.SiteAds, error) {
	ret := _m.Called(ctx, siteIDs)

	if rf, ok := ret.Get(0).(func(context.Context, []string) ([]*model.SiteAds, error)); ok {
		return rf(ctx, siteIDs)
	}

	var r0 []*model.SiteAds
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.SiteAds)
	}
	return r0, ret.Error(1)
}

// Upsert provides a mock function with given fields: ctx, ads
func (_m *ISiteAdsRepository) Upsert(ctx context.Context, ads *model.SiteAds) error {
	ret := _m.Called(ctx, ads)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *model.SiteAds) error); ok {
		r0 = rf(ctx, ads)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewISiteAdsRepository creates a new instance of ISiteAdsRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewISiteAdsRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *ISiteAdsRepository {
	m := &ISiteAdsRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
