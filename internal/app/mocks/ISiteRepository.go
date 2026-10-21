// Written by hand in the layout mockery (v2) generates, because mockery is
// not installed on this machine. Regenerate with mockery once it is
// (see README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// ISiteRepository is a mock type for the repository.ISiteRepository type
type ISiteRepository struct {
	mock.Mock
}

// GetByIDs provides a mock function with given fields: ctx, siteIDs
func (_m *ISiteRepository) GetByIDs(ctx context.Context, siteIDs []string) ([]*model.Site, error) {
	ret := _m.Called(ctx, siteIDs)

	var r0 []*model.Site
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.Site)
	}
	return r0, ret.Error(1)
}

// GetIDsBySearch provides a mock function with given fields: ctx, search
func (_m *ISiteRepository) GetIDsBySearch(ctx context.Context, search string) ([]string, error) {
	ret := _m.Called(ctx, search)

	var r0 []string
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]string)
	}
	return r0, ret.Error(1)
}

// NewISiteRepository creates a new instance of ISiteRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewISiteRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *ISiteRepository {
	m := &ISiteRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
