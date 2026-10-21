// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// ISiteContextRepository is a mock type for the repository.ISiteContextRepository type
type ISiteContextRepository struct {
	mock.Mock
}

// GetBySiteID provides a mock function with given fields: ctx, siteID
func (_m *ISiteContextRepository) GetBySiteID(ctx context.Context, siteID string) (*model.SiteContext, error) {
	ret := _m.Called(ctx, siteID)

	if rf, ok := ret.Get(0).(func(context.Context, string) (*model.SiteContext, error)); ok {
		return rf(ctx, siteID)
	}

	var r0 *model.SiteContext
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.SiteContext)
	}
	return r0, ret.Error(1)
}

// GetCompetitorsBySiteID provides a mock function with given fields: ctx, siteID
func (_m *ISiteContextRepository) GetCompetitorsBySiteID(ctx context.Context, siteID string) ([]*model.SiteCompetitor, error) {
	ret := _m.Called(ctx, siteID)

	if rf, ok := ret.Get(0).(func(context.Context, string) ([]*model.SiteCompetitor, error)); ok {
		return rf(ctx, siteID)
	}

	var r0 []*model.SiteCompetitor
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.SiteCompetitor)
	}
	return r0, ret.Error(1)
}

// GetCustomNotesBySiteID provides a mock function with given fields: ctx, siteID
func (_m *ISiteContextRepository) GetCustomNotesBySiteID(ctx context.Context, siteID string) ([]*model.SiteCustomNote, error) {
	ret := _m.Called(ctx, siteID)

	if rf, ok := ret.Get(0).(func(context.Context, string) ([]*model.SiteCustomNote, error)); ok {
		return rf(ctx, siteID)
	}

	var r0 []*model.SiteCustomNote
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.SiteCustomNote)
	}
	return r0, ret.Error(1)
}

// NewISiteContextRepository creates a new instance of ISiteContextRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewISiteContextRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *ISiteContextRepository {
	m := &ISiteContextRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
